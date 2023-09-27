package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cdevr/cpush/cisco"
	"github.com/cdevr/cpush/configfile"
	"github.com/cdevr/cpush/pwcache"
	"github.com/cdevr/cpush/shell"
	"github.com/cdevr/cpush/utils"

	"golang.org/x/net/proxy"
)

//go:generate go run tagBuild.go

var device = flag.String("device", "", "a device to execute commands on")
var deviceFile = flag.String("devicefile", "", "file with a list of device to execute commands on. One device per line")
var deviceStdIn = flag.Bool("devicestdin", false, "read list of devices from stdin (don't forget to CTRL-D, or provide EOF)")
var deviceList = flag.String("devices", "", "comma-separated list of routers")

var command = flag.String("cmd", "", "a command to execute")
var push = flag.String("push", "", "something put into the configuration")
var interactive = flag.Bool("i", false, "create an interactive shell on the device")

var suppressBanner = flag.Bool("suppress_banner", true, "suppress the SSH banner and login")
var suppressAdmin = flag.Bool("suppress_admin", true, "suppress administrative information")
var suppressSending = flag.Bool("suppress_sending", true, "suppress what is being sent to the router")
var suppressOutput = flag.Bool("suppress_output", false, "don't print router output")
var showDeviceName = flag.Bool("devicename", true, "prefix output from routers with the device name")
var logOutputTemplate = flag.String("output", "", "template for files to save the output in. %s gets replaced with the device name")

var version = flag.Bool("version", false, "print version and exit")

var username = flag.String("username", "", "username to use for login")

var retries = flag.Int("retries", 3, "retries (per device)")
var timeout = flag.Duration("timeout", 10*time.Second, "timeout for the command")

var cacheAllowed = flag.Bool("pw_cache_allowed", true, "allowed to cache password in /dev/shm")
var clearPwCache = flag.Bool("pw_clear_cache", false, "forcibly clear the pw cache")

var socks = flag.String("socks", "", "proxy to use")

func GetUser() string {
	cur, err := user.Current()
	if err != nil {
		log.Fatalf("Cannot get current user")
	}
	username := cur.Username
	if strings.HasPrefix(username, "adm1-") {
		username = username[5:]
	}
	return username
}

type routerOutput struct {
	router string
	output string
}

type routerError struct {
	router string
	err    error
}

// CmdDevices executes a command on many devices, prints the output.
func CmdDevices(opts *cisco.Options, devices []string, username string, password string, cmd string) {
	var wg sync.WaitGroup

	outputs := make(chan routerOutput)
	errors := make(chan routerError)

	done := make(chan bool)

	started := make(chan bool)
	ended := make(chan bool)

	startCount := 0
	endedCount := 0

	doDevice := func(device string) {
		defer wg.Done()
		started <- true
		defer func() { ended <- true }()

		var output string
		var err error
		done := make(chan bool)
		go func() {
			output, err = cisco.Cmd(opts, device, username, password, cmd)
			done <- true
		}()

		select {
		case <-done:
			if err != nil {
				errors <- routerError{device, err}
			} else {
				outputs <- routerOutput{device, output}
			}
		case <-time.After(*timeout):
			errors <- routerError{device, fmt.Errorf("router %q hit timeout after %v", device, *timeout)}
		}
	}

	for _, d := range devices {
		wg.Add(1)
		go doDevice(d)
	}

	go func() {
		wg.Wait()
		done <- true
	}()

	allDone := false
	for !allDone {
		select {
		case <-started:
			startCount += 1
			fmt.Printf("\033[2K\r%d/%d/%d", startCount, endedCount, len(devices))
		case <-ended:
			startCount -= 1
			endedCount += 1
			fmt.Printf("\033[2K\r%d/%d/%d", startCount, endedCount, len(devices))
		case re := <-errors:
			fmt.Printf("\rerror on %q: %v\n", re.router, re.err)
		case output := <-outputs:
			if *logOutputTemplate != "" {
				fn := strings.ReplaceAll(*logOutputTemplate, "%s", output.router)
				err := utils.ReplaceFile(fn, utils.Dos2Unix(output.output))
				if err != nil {
					log.Printf("failed to save output for router %q: %v", output.router, err)
				}
			}

			lines := strings.Split(output.output, "\n")
			for _, line := range lines {
				if !*suppressOutput {
					if *showDeviceName {
						fmt.Printf("%s: %s\n", output.router, line)
					} else {
						fmt.Printf("%s\n", line)
					}
				}
			}
		case <-done:
			allDone = true
			fmt.Printf("\033[2K\r%d/%d/%d", startCount, endedCount, len(devices))
			fmt.Printf("\n")
		}
	}
}

// filterEmptyDevices trims spaces and removes empty string from a list of strings.
func filterEmptyDevices(devices []string) []string {
	filteredDevices := []string{}
	for _, device := range devices {
		device = strings.TrimSpace(device)
		if len(device) > 0 {
			filteredDevices = append(filteredDevices, device)
		}
	}
	return filteredDevices
}

func main() {
	configfile.ParseConfigFile("~/.cpush")
	flag.Parse()

	if *version {
		fmt.Printf("cpush git revision %s compiled at %s with Go %s\n", buildGitRevision, buildTime, runtime.Version())
		return
	}

	if len(os.Args) == 1 {
		fmt.Printf(`cpush tool to send commands to Cisco and Juniper routers
	
Simplest usage:

	cpush device command goes here.

Example:

	cpush rtr1 show version

Other flags are:`)
		flag.Usage()
		return
	}
	// Allow device and command arguments to be passed in as non-args.
	if *device == "" && *deviceList == "" && *deviceFile == "" && flag.NArg() == 1 && *command == "" && *push == "" {
		*device = flag.Arg(0)
		*interactive = true
	} else if *device == "" && *command == "" && flag.NArg() >= 2 {
		*device = flag.Arg(0)
		*command = strings.Join(flag.Args()[1:], " ")
	}

	if *command == "" && *push == "" && *interactive == false {
		log.Printf("you didn't pass in a command or a confliglet")
		return
	}
	if *username == "" {
		*username = GetUser()
	}

	var dialer proxy.Dialer = proxy.Direct
	if *socks != "" {
		var err error
		dialer, err = proxy.SOCKS5("tcp", *socks, nil, nil)
		if err != nil {
			log.Fatalf("failed to dial SOCKS server at %q: %v", *socks, err)
		}
	}

	opts := cisco.NewOptions()
	opts.SuppressAdmin(*suppressAdmin).SuppressBanner(*suppressBanner).SuppressSending(*suppressSending).Timeout(*timeout).Dialer(dialer.Dial).SuppressOutput(*suppressOutput)

	password, err := pwcache.GetPassword(*cacheAllowed, *clearPwCache)
	if err != nil {
		log.Fatalf("error getting password for user: %v", err)
	}
	if *clearPwCache {
		return
	}

	if *device != "" {
		if *interactive {
			err = shell.Interactive(opts, *device, *username, password)
			if err != nil {
				log.Fatalf("failed to start interactive shell: %v", err)
			}
			return
		}
		var output string
		if *command != "" {
			output, err = cisco.Cmd(opts, *device, *username, password, *command)
			if err != nil {
				log.Fatalf("failed to execute command %q on device %q: %v", *command, *device, err)
			}
		}
		if *push != "" {
			output, err = cisco.Push(opts, *device, *username, password, *push)
			if err != nil {
				log.Fatalf("failed to commit configlet %q on device %q: %v", *command, *device, err)
			}
		}
		if *logOutputTemplate != "" {
			fn := strings.ReplaceAll(*logOutputTemplate, "%s", *device)
			err := utils.AppendToFile(fn, utils.Dos2Unix(output))
			if err != nil {
				log.Printf("failed to save output for router %q: %v", *device, err)
			}
		}
		if !*suppressOutput {
			fmt.Printf("%s\n", output)
		}
	} else if *deviceList != "" {
		devices := strings.Split(*deviceList, ",")

		CmdDevices(opts, filterEmptyDevices(devices), *username, password, *command)
	} else if *deviceFile != "" {
		fileLines, err := os.ReadFile(*deviceFile)
		if err != nil {
			log.Fatalf("failed to read device file %q: %v", *deviceFile, err)
		}
		devices := strings.Split(string(fileLines), "\n")

		CmdDevices(opts, filterEmptyDevices(devices), *username, password, *command)
	} else if *deviceStdIn {
		fileLines, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("failed to read devices from stdin %q: %v", *deviceFile, err)
		}
		devices := strings.Split(string(fileLines), "\n")

		CmdDevices(opts, devices, *username, password, *command)
	}
	os.Exit(0)
}
