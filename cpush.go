package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"strings"
	"sync"
	"time"

	"github.com/cdevr/cpush/cisco"
	"github.com/cdevr/cpush/pwcache"
	"github.com/cdevr/cpush/utils"
	"github.com/cdevr/cpush/configfile"

	"golang.org/x/net/proxy"
)

var device = flag.String("device", "", "a device to execute commands on")
var deviceFile = flag.String("devicefile", "", "file with a list of device to execute commands on. One device per line")
var deviceStdIn = flag.Bool("devicestdin", false, "read list of devices from stdin (don't forget to CTRL-D, or provide EOF)")
var deviceList = flag.String("devices", "", "comma-separated list of routers")
var command = flag.String("cmd", "", "a command to execute")

var suppressBanner = flag.Bool("suppress_banner", true, "suppress the SSH banner and login")
var suppressAdmin = flag.Bool("suppress_admin", true, "suppress administrative information")
var suppressSending = flag.Bool("suppress_sending", true, "suppress what is being sent to the router")
var showDeviceName = flag.Bool("devicename", true, "prefix output from routers with the device name")
var logOutputTemplate = flag.String("output", "", "template for files to save the output in. %s gets replaced with the device name")

var username = flag.String("username", "", "username to use for login")

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

// CmdDevices executes a command on many devices, prints the output.
func CmdDevices(opts *cisco.Options, devices []string, username string, password string, cmd string) {
	var wg sync.WaitGroup

	errors := make(chan error)
	outputs := make(chan routerOutput)
	done := make(chan bool)

	for _, d := range devices {
		wg.Add(1)
		go func(device string) {
			defer wg.Done()

			var output string
			var err error
			done := make(chan bool)
			go func() {
				output, err = cisco.Cmd(opts, device, username, password, cmd)
				if err != nil {
					errors <- fmt.Errorf("failed to execute command %q on device %q: %v", cmd, device, err)
				}
				done <- true
			}()

			select {
			case <-done:
				outputs <- routerOutput{device, output}
			case <-time.After(*timeout):
				errors <- fmt.Errorf("router %q hit timeout after %v", device, *timeout)
			}
		}(d)
	}

	go func() {
		wg.Wait()
		done <- true
	}()

	allDone := false
	for !allDone {
		select {
		case err := <-errors:
			fmt.Printf("error: %v\n", err)
		case output := <-outputs:
			lines := strings.Split(output.output, "\n")
			for _, line := range lines {
				if *logOutputTemplate != "" {
					fn := strings.ReplaceAll(*logOutputTemplate, "%s", output.router)
					err := utils.AppendToFile(fn, line)
					if err != nil {
						log.Printf("failed to save output for router %q: %v", output.router, err)
					}
				}
				if *showDeviceName {
					fmt.Printf("%s: %s\n", output.router, line)
				} else {
					fmt.Printf("%s\n", line)
				}
			}
		case <-done:
			allDone = true
		}
	}
}

func main() {
	configfile.ParseConfigFile("~/.cpush")
	flag.Parse()

	if flag.NArg()+flag.NFlag() == 0 {
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
	if *device == "" && *command == "" && flag.NArg() >= 2 {
		*device = flag.Arg(0)
		*command = strings.Join(flag.Args()[1:], " ")
	}

	if *device == "" && *deviceList == "" && *deviceFile == "" {
		*deviceStdIn = true
	}
	if *command == "" {
		log.Printf("you didn't pass in a device")
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
	opts.SuppressAdmin(*suppressAdmin).SuppressBanner(*suppressBanner).SuppressSending(*suppressSending).Timeout(*timeout).Dialer(dialer.Dial)

	password, err := pwcache.GetPassword(*cacheAllowed, *clearPwCache)
	if err != nil {
		log.Fatalf("error getting password for user: %v", err)
	}
	if *clearPwCache {
		return
	}

	if *device != "" {
		output, err := cisco.Cmd(opts, *device, *username, password, *command)
		if err != nil {
			log.Fatalf("failed to execute command %q on device %q: %v", *command, *device, err)
		}
		fmt.Printf("%s\n", output)
	} else if *deviceList != "" {
		devices := strings.Split(*deviceList, ",")

		CmdDevices(opts, devices, *username, password, *command)
	} else if *deviceFile != "" {
		fileLines, err := os.ReadFile(*deviceFile)
		if err != nil {
			log.Fatalf("failed to read device file %q: %v", *deviceFile, err)
		}
		devices := strings.Split(string(fileLines), "\n")

		CmdDevices(opts, devices, *username, password, *command)
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
