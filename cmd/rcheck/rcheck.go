// cpush execute commands on cisco routers and push configuration.
package main

import (
	"flag"
	"fmt"
	"github.com/cdevr/cpush/checks"
	"github.com/cdevr/cpush/options"
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
	"golang.org/x/net/proxy"
)

//go:generate go run tagBuild.go

var (
	device      = flag.String("device", "", "a device to execute commands on")
	deviceFile  = flag.String("devicefile", "", "file with a list of device to execute commands on. One device per line")
	deviceStdIn = flag.Bool("devicestdin", false, "read list of devices from stdin (don't forget to CTRL-D, or provide EOF)")
	deviceList  = flag.String("devices", "", "comma-separated list of routers")

	suppressBanner   = flag.Bool("suppress_banner", true, "suppress the SSH banner and login")
	suppressAdmin    = flag.Bool("suppress_admin", true, "suppress administrative information")
	suppressSending  = flag.Bool("suppress_sending", true, "suppress what is being sent to the router")
	suppressOutput   = flag.Bool("suppress_output", false, "don't print router output")
	suppressProgress = flag.Bool("suppress_progress", false, "don't show progress indicator")

	_ = flag.Bool("devicename", true, "prefix output from routers with the device name")
	_ = flag.String("output", "", "template for files to save the output in. %s gets replaced with the device name")

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

type routerError struct {
	router string
	err    error
}

func CheckRouters(opts *options.Options, concurrentLimit int, devices []string, username string, password string) {
	checkCommands := checks.GetCheckCommands()

	var wg sync.WaitGroup

	deviceChan := make(chan string)

	outputs := make(chan []checks.CheckResult)
	errors := make(chan routerError)

	done := make(chan bool)

	started := make(chan bool)
	ended := make(chan bool)

	startCount := 0
	endedCount := 0

	checkDevice := func(device string) ([]checks.CheckResult, error) {
		cmdResults := map[string]string{}

		for _, cmd := range checkCommands {
			output, err := cisco.Cmd(opts, device, username, password, cmd)
			if err != nil {
				return nil, err
			}
			cmdResults[cmd] = output
		}

		return checks.Check(device, cmdResults)
	}

	worker := func() {
		defer wg.Done()
	devices:
		for device := range deviceChan {
			for itry := 0; itry < *retries; itry += 1 {
				output, err := checkDevice(device)
				if err != nil {
					errors <- routerError{device, err}
				} else {
					outputs <- output
					continue devices
				}
				fmt.Fprintf(os.Stderr, "Retrying %q: %d/%d", device, itry, *retries)
			}
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
				fmt.Fprintf(os.Stderr, "\033[2K\r%d/%d/%d/%d", remaining, startCount, endedCount, len(devices))
			}
		case <-ended:
			startCount -= 1
			endedCount += 1
			if !*suppressProgress {
				fmt.Fprintf(os.Stderr, "\033[2K\r%d/%d/%d/%d", remaining, startCount, endedCount, len(devices))
			}
		case re := <-errors:
			fmt.Fprintf(os.Stderr, "\rerror on %q: %v\n", re.router, re.err)
		case output := <-outputs:
			for _, cr := range output {
				fmt.Printf("%s %s: %s\n", cr.CheckName, cr.Device, cr.Result)
			}
		case <-done:
			allDone = true
			if !*suppressProgress {
				fmt.Fprintf(os.Stderr, "\033[2K\r%d/%d/%d/%d", remaining, startCount, endedCount, len(devices))
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
		fmt.Printf(`rcheck tool to check router state`)
		flag.Usage()
		return
	}
	// Allow device and command arguments to be passed in as non-args.
	if *device == "" && *deviceList == "" && *deviceFile == "" {
		*device = flag.Arg(0)
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

	var devices []string

	if *device != "" {
		devices = []string{*device}
	} else if *deviceList != "" {
		devices = filterEmptyDevices(strings.Split(*deviceList, ","))
	} else if *deviceFile != "" {
		fileLines, err := os.ReadFile(*deviceFile)
		if err != nil {
			log.Fatalf("failed to read device file %q: %v", *deviceFile, err)
		}
		devices = filterEmptyDevices(strings.Split(string(fileLines), "\n"))
	} else if *deviceStdIn {
		fileLines, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("failed to read devices from stdin %q: %v", *deviceFile, err)
		}
		devices = filterEmptyDevices(strings.Split(string(fileLines), "\n"))
	}

	CheckRouters(opts, *concurrentLimit, devices, *username, password)
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
