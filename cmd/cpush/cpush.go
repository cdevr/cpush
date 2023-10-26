// cpush execute commands on cisco routers and push configuration.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cdevr/cpush/options"

	"github.com/cdevr/cpush/cisco"
	"github.com/cdevr/cpush/configfile"
	"github.com/cdevr/cpush/pwcache"
	"github.com/cdevr/cpush/shell"
	"github.com/cdevr/cpush/utils"

	"golang.org/x/net/proxy"
)

//go:generate go run tagBuild.go

// ANSI code to erase the current line and put the cursor at the beginning.
const clearLine = "\033[2K\r"

var (
	device = flag.String("device", "", "a device to execute commands on")

	command     = flag.String("cmd", "", "a command to execute")
	push        = flag.String("push", "", "something put into the configuration. If it has file: prefix, it will be read from that file")
	interactive = flag.Bool("i", false, "create an interactive shell on the device")

	suppressBanner   = flag.Bool("suppress_banner", true, "suppress the SSH banner and login")
	suppressAdmin    = flag.Bool("suppress_admin", true, "suppress administrative information")
	suppressSending  = flag.Bool("suppress_sending", true, "suppress what is being sent to the router")
	suppressOutput   = flag.Bool("suppress_output", false, "don't print router output")
	suppressProgress = flag.Bool("suppress_progress", false, "don't show progress indicator")

	showDeviceName    = flag.Bool("devicename", true, "prefix output from routers with the device name")
	logOutputTemplate = flag.String("output", "", "template for files to save the output in. %s gets replaced with the device name")

	version = flag.Bool("version", false, "print version and exit")

	username = flag.String("username", "", "username to use for login")

	retries         = flag.Int("retries", 3, "retries (per device)")
	timeout         = flag.Duration("timeout", 10*time.Second, "timeout for the command")
	concurrentLimit = flag.Int("limit", 25, "maximum number of simultaneous devices")

	usePwCache   = flag.Bool("pw_cache_allowed", true, "allowed to cache password in /dev/shm")
	clearPwCache = flag.Bool("pw_clear_cache", false, "forcibly clear the pw cache")

	socks = flag.String("socks", "", "proxy to use")
)

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
func CmdDevices(opts *options.Options, concurrentLimit int, devices []string, username string, password string, cmd string) {
	var wg sync.WaitGroup

	deviceChan := make(chan string)

	outputs := make(chan routerOutput)
	errors := make(chan routerError)

	done := make(chan bool)

	started := make(chan bool)
	ended := make(chan bool)

	startCount := 0
	endedCount := 0

	doDevice := func(device string) (string, error) {
		started <- true
		defer func() { ended <- true }()

		var output string
		var err error
		done := make(chan bool)
		go func() {
			output, err = cisco.Cmd(opts, device, username, password, cmd, *timeout)
			done <- true
		}()

		select {
		case <-done:
			return output, err
		case <-time.After(*timeout):
			return "", fmt.Errorf("router %q hit timeout after %v", device, *timeout)
		}
	}

	worker := func() {
		defer wg.Done()
	devices:
		for device := range deviceChan {
			var err error

			for itry := 0; itry < *retries; itry += 1 {
				var output string
				output, err = doDevice(device)
				if err == nil {
					outputs <- routerOutput{device, output}
					continue devices
				}
				fmt.Fprintf(os.Stderr, clearLine+"Retrying %q: %d/%d\n", device, itry, *retries)
			}

			errors <- routerError{device, err}
		}
	}

	for i := 0; i < concurrentLimit; i++ {
		wg.Add(1)
		go worker()
	}

	go func() {
		for _, d := range devices {
			deviceChan <- d
		}
		close(deviceChan)
	}()

	go func() {
		wg.Wait()
		done <- true
	}()

	allDone := false
	for !allDone {
		remaining := len(devices) - startCount - endedCount
		select {
		case <-started:
			startCount += 1
			if !*suppressProgress {
				fmt.Fprintf(os.Stderr, clearLine+"%d/%d/%d/%d", remaining, startCount, endedCount, len(devices))
			}
		case <-ended:
			startCount -= 1
			endedCount += 1
			if !*suppressProgress {
				fmt.Fprintf(os.Stderr, clearLine+"%d/%d/%d/%d", remaining, startCount, endedCount, len(devices))
			}
		case re := <-errors:
			fmt.Fprintf(os.Stderr, clearLine+"error on %q: %v\n", re.router, re.err)
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
			if !*suppressProgress {
				fmt.Fprintf(os.Stderr, clearLine+"%d/%d/%d/%d", remaining, startCount, endedCount, len(devices))
				fmt.Fprintf(os.Stderr, "\n")
			}
		}
	}
}

// filterEmptyDevices trims spaces and removes empty string from a list of strings.
func filterEmptyDevices(devices []string) []string {
	var filteredDevices []string
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
	if *device == "" && flag.NArg() == 1 && *command == "" && *push == "" {
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

	dialer := MakeDialer(*socks)

	opts := options.NewOptions()
	opts.SuppressAdmin = *suppressAdmin
	opts.SuppressBanner = *suppressBanner
	opts.SuppressSending = *suppressSending
	opts.SuppressOutput = *suppressOutput
	opts.Timeout = *timeout
	opts.Dialer = dialer.Dial

	password, err := pwcache.GetPassword(*clearPwCache, *usePwCache)
	if err != nil {
		log.Fatalf("error getting password for user: %v", err)
	}
	if *clearPwCache {
		return
	}

	if strings.Contains(*device, ",") {
		devices := strings.Split(*device, ",")

		CmdDevices(opts, *concurrentLimit, filterEmptyDevices(devices), *username, password, *command)
	} else if strings.HasPrefix(*device, "file:") {
		deviceFn := (*device)[5:]
		fileLines, err := os.ReadFile(deviceFn)
		if err != nil {
			log.Fatalf("failed to read device file %q: %v", deviceFn, err)
		}
		devices := strings.Split(string(fileLines), "\n")

		CmdDevices(opts, *concurrentLimit, filterEmptyDevices(devices), *username, password, *command)
	} else if *device != "" {
		if *interactive {
			err = shell.Interactive(opts, *device, *username, password)
			if err != nil {
				log.Fatalf("failed to start interactive shell: %v", err)
			}
			return
		}
		var output string
		if *command != "" {
			output, err = cisco.Cmd(opts, *device, *username, password, *command, *timeout)
			if err != nil {
				log.Fatalf("failed to execute command %q on device %q: %v", *command, *device, err)
			}
		}
		if *push != "" {
			topush := *push
			if strings.HasPrefix(topush, "file:") {
				fn := topush[5:]
				topushBytes, err := os.ReadFile(fn)
				if err != nil {
					log.Fatalf("failed to read push lines from %q: %v", fn, err)
				}
				topush = string(topushBytes)
			}
			log.Printf("pushing to %q: %q", *device, topush)
			output, err = cisco.Push(opts, *device, *username, password, topush)
			if err != nil {
				log.Fatalf("failed to commit configlet %q on device %q: %v", topush, *device, err)
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
	}
	os.Exit(0)
}

func MakeDialer(proxyAddress string) proxy.Dialer {
	var dialer proxy.Dialer = proxy.Direct
	if proxyAddress != "" {
		var err error
		dialer, err = proxy.SOCKS5("tcp", proxyAddress, nil, nil)
		if err != nil {
			log.Fatalf("failed to make dialer for proxy server at %q: %v", proxyAddress, err)
		}
	}
	return dialer
}
