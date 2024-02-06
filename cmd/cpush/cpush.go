// cpush execute commands on cisco routers and push configuration.
package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/user"
	"path"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cdevr/cpush/options"
	"github.com/cdevr/cpush/texttable"

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

	showDeviceName = flag.Bool("devicename", true, "prefix output from routers with the device name")

	outputFile         = flag.String("output", "", "template for files to save the output in. %s gets replaced with the device name. When specified output is not printed")
	skipIfOutputExists = flag.Bool("skip_if_output_exists", true, "skip the device if the output file already exists")

	version = flag.Bool("version", false, "print version and exit")

	username = flag.String("username", "", "username to use for login")

	retries         = flag.Int("retries", 3, "retries (per device)")
	timeout         = flag.Duration("timeout", 30*time.Second, "timeout for the command")
	concurrentLimit = flag.Int("limit", 25, "maximum number of simultaneous devices")

	usePwCache   = flag.Bool("pw_cache_allowed", true, "allowed to cache password in /dev/shm")
	clearPwCache = flag.Bool("pw_clear_cache", false, "forcibly clear the pw cache")

	shuffle = flag.Bool("shuffle", false, "if true, and doing multiple devices, randomize the order")

	socks = flag.String("socks", "", "proxy to use")
)

func GetUser() string {
	cur, err := user.Current()
	if err != nil {
		log.Fatalf("Cannot get current user")
	}
	username := cur.Username
	username = strings.TrimPrefix(username, "adm1-")

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

// FileExists returns true if a file with the given name exists.
func FileExists(fn string) bool {
	_, err := os.Stat(fn)
	return err == nil
}

// FillOutputFilenameTemplate fills in the router name in the filename template.
func FillOutputFilenameTemplate(fn string, router string) string {
	return strings.ReplaceAll(fn, "%s", router)
}

type DoFunc func(opts *options.Options, device string, username string, password string, param string, timeout time.Duration) (string, error)

// DoManyDevices executes a push or a command on many devices, prints the output.
func DoManyDevices(opts *options.Options, concurrentLimit int, devices []string, username string, password string, param string, shuffle bool, do DoFunc) {
	var startTime = time.Now()
	var wg sync.WaitGroup

	deviceChan := make(chan string)

	outputs := make(chan routerOutput)
	errors := make(chan routerError)

	done := make(chan bool)

	start := make(chan string)
	end := make(chan string)

	retry := make(chan string)

	skippedCount := 0
	startedCount := 0
	endedCount := 0

	succeeded := map[string]bool{}
	failed := map[string]bool{}

	doDevice := func(device string) (string, error) {

		var output string
		var err error
		done := make(chan bool)
		go func() {
			output, err = do(opts, device, username, password, param, *timeout)
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
			start <- device

			var err error

			for iTry := 0; iTry < *retries; iTry += 1 {
				var output string
				output, err = doDevice(device)
				if err == nil {
					outputs <- routerOutput{device, output}
					end <- device
					continue devices
				}
				retry <- fmt.Sprintf("Retrying %q: %d/%d", device, iTry+1, *retries)
			}

			end <- device
			errors <- routerError{device, fmt.Errorf("failed in %d tries, last error: %v", *retries, err)}
		}
	}

	for i := 0; i < concurrentLimit; i++ {
		wg.Add(1)
		go worker()
	}

	go func() {
		// Shuffle devices to do them in random order if requested.
		if shuffle {
			rand.Shuffle(len(devices), func(i, j int) { devices[i], devices[j] = devices[j], devices[i] })
		}

		// Skip all the devices we're going to skip.
		var dontSkip []string
		for _, d := range devices {
			// Skip this device if the output file already exists.
			if *skipIfOutputExists {
				if *outputFile != "" && FileExists(FillOutputFilenameTemplate(*outputFile, d)) {
					log.Printf("skipping %q: %q already exists", d, FillOutputFilenameTemplate(*outputFile, d))
					skippedCount += 1
					continue
				}
			}
			dontSkip = append(dontSkip, d)
		}

		// Then execute the remainder.
		for _, d := range dontSkip {
			deviceChan <- d
		}
		close(deviceChan)
	}()

	go func() {
		wg.Wait()
		done <- true
	}()

	progressLine := func() string {
		inProgress := startedCount - endedCount
		remaining := len(devices) - inProgress - endedCount - skippedCount
		progress := float64(endedCount) / float64(len(devices)-skippedCount)

		timeElapsed := time.Since(startTime).Round(time.Second)
		expectedDuration := time.Duration(float64(timeElapsed) / progress).Round(time.Second)

		expectedFinish := time.Now().Add(expectedDuration).Round(time.Second)

		expectedDurationStr := expectedDuration.String()
		expectedFinishStr := expectedFinish.String()
		if timeElapsed < 2*time.Second || endedCount < 1 {
			timeElapsed = 0
			expectedDurationStr = "..."
			expectedFinishStr = "..."
		}

		return fmt.Sprintf("%d/%d/%d/%d %2.2f%% %s/%s expected finish @ %v", remaining, inProgress, endedCount+skippedCount, len(devices), 100.0*progress, timeElapsed, expectedDurationStr, expectedFinishStr)
	}

	allDone := false
	for !allDone {
		select {
		case <-start:
			startedCount += 1
			if !*suppressProgress {
				fmt.Fprint(os.Stderr, clearLine+progressLine())
			}
		case <-end:
			endedCount += 1
			if !*suppressProgress {
				fmt.Fprint(os.Stderr, clearLine+progressLine())
			}
		case msg := <-retry:
			fmt.Fprintf(os.Stderr, clearLine+msg+"\n")
			fmt.Fprint(os.Stderr, clearLine+progressLine())
		case re := <-errors:
			failed[re.router] = true
			fmt.Fprintf(os.Stderr, clearLine+"error on %q: %v\n", re.router, re.err)
			fmt.Fprint(os.Stderr, clearLine+progressLine())
		case rtrOutput := <-outputs:
			succeeded[rtrOutput.router] = true
			if *outputFile != "" {
				fn := FillOutputFilenameTemplate(*outputFile, rtrOutput.router)
				err := utils.ReplaceFile(fn, utils.Dos2Unix(rtrOutput.output))
				if err != nil {
					log.Printf("failed to save output for router %q: %v", rtrOutput.router, err)
				}
			}

			lines := strings.Split(rtrOutput.output, "\n")
			written := false
			for _, line := range lines {
				if !*suppressOutput {
					if *showDeviceName {
						fmt.Fprint(os.Stderr, clearLine)
						fmt.Printf("%s: %s\n", rtrOutput.router, line)
					} else {
						fmt.Fprint(os.Stderr, clearLine)
						fmt.Printf("%s\n", line)
					}
					written = true
				}
			}
			if written {
				os.Stdout.Sync()
			}
			fmt.Fprint(os.Stderr, clearLine+progressLine())
		case <-done:
			allDone = true
			if !*suppressProgress {
				fmt.Fprint(os.Stderr, clearLine+progressLine())
				fmt.Fprintf(os.Stderr, "\n")
			}
		}
	}

	PrintSummary(succeeded, failed)
}

// PrintSummary will print an overview of succeeded and failed devices.
func PrintSummary(succeeded map[string]bool, failed map[string]bool) {
	var sortedSucceeded []string
	for rtr := range succeeded {
		sortedSucceeded = append(sortedSucceeded, rtr)
	}
	sort.Strings(sortedSucceeded)

	var sortedFailed []string
	for rtr := range failed {
		sortedFailed = append(sortedFailed, rtr)
	}
	sort.Strings(sortedFailed)

	// Print summary
	fmt.Fprintf(os.Stderr, "\nSucceeded (%d devices)\n\n", len(sortedSucceeded))

	if len(sortedSucceeded) == 0 {
		fmt.Fprintln(os.Stderr, "(None)")
	} else {
		fmt.Fprintln(os.Stderr, texttable.Columns(sortedSucceeded, 4))
	}
	fmt.Fprintf(os.Stderr, "\n")

	fmt.Fprintf(os.Stderr, "Failed (%d devices)\n\n", len(sortedFailed))

	if len(sortedFailed) == 0 {
		fmt.Fprintln(os.Stderr, "(None)")
	} else {
		fmt.Fprintln(os.Stderr, texttable.Columns(sortedFailed, 4))
	}
	fmt.Fprintln(os.Stderr)
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

func isFlagPresent(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// ResolveFile will read a file path, and if it starts with "file:" will replace it with the contents
func ResolveFile(filepath string) (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("error getting user: %w", err)
	}
	dir := user.HomeDir

	if strings.HasPrefix(filepath, "~/") {
		filepath = path.Join(dir, filepath[2:])
	}

	if path.IsAbs(filepath) {
		return filepath, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("error getting current directory: %w", err)
	}

	filepath = path.Join(cwd, filepath)

	return filepath, nil
}

// ResolveFilePrefix will read a router specification,
// and if it starts with "file:" will replace it with the
// contents of the file specified.
func ResolveFilePrefix(s string) (string, error) {
	if fn := strings.TrimPrefix(s, "file:"); fn != s {
		rfn, err := ResolveFile(fn)
		if err != nil {
			return "", err
		}
		bts, err := os.ReadFile(rfn)
		if err != nil {
			return "", err
		}
		return string(bts), nil
	}
	return s, nil
}

func main() {
	configfile.ParseConfigFile("~/.cpush")
	flag.Parse()

	// Set suppress_output flag if output flag was passed, unless override.
	if !isFlagPresent("suppress_output") && isFlagPresent("output") {
		*suppressOutput = true
	}

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

	password, err := pwcache.GetPassword(*clearPwCache, *usePwCache)
	if err != nil {
		log.Fatalf("error getting password for user: %v", err)
	}
	if *clearPwCache {
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

	if *command == "" && *push == "" && !*interactive {
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
	opts.Dialer = dialer.DialContext

	toPush, err := ResolveFilePrefix(*push)
	if err != nil {
		log.Fatalf("error resolving %q: %v", *push, err)
	}

	if strings.Contains(*device, ",") {
		devices := strings.Split(*device, ",")

		if *command != "" {
			DoManyDevices(opts, *concurrentLimit, filterEmptyDevices(devices), *username, password, *command, *shuffle, cisco.Cmd)
		} else if toPush != "" {
			DoManyDevices(opts, *concurrentLimit, filterEmptyDevices(devices), *username, password, toPush, *shuffle, cisco.Push)
		} else {
			fmt.Fprint(os.Stderr, "nothing to do")
		}
	} else if strings.HasPrefix(*device, "file:") {
		deviceFn := (*device)[5:]
		fileLines, err := os.ReadFile(deviceFn)
		if err != nil {
			log.Fatalf("failed to read devices list file %q: %v", deviceFn, err)
		}
		devices := strings.Split(string(fileLines), "\n")

		if *command != "" {
			DoManyDevices(opts, *concurrentLimit, filterEmptyDevices(devices), *username, password, *command, *shuffle, cisco.Cmd)
		} else if toPush != "" {
			DoManyDevices(opts, *concurrentLimit, filterEmptyDevices(devices), *username, password, toPush, *shuffle, cisco.Push)
		} else {
			fmt.Fprint(os.Stderr, "nothing to do")
		}
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
		} else if toPush != "" {
			output, err = cisco.Push(opts, *device, *username, password, toPush, *timeout)
			if err != nil {
				log.Fatalf("failed to push configlet %q on device %q: %v", toPush, *device, err)
			}
		} else {
			fmt.Fprint(os.Stderr, "nothing to do")
		}

		if *outputFile != "" {
			fn := strings.ReplaceAll(*outputFile, "%s", *device)
			err := utils.AppendToFile(fn, utils.Dos2Unix(output))
			if err != nil {
				log.Printf("failed to save output for router %q: %v", *device, err)
			}
		}

		if !*suppressOutput {
			fmt.Printf("%s\n", output)
		}
	}
}

func MakeDialer(proxyAddress string) proxy.ContextDialer {
	var dialer proxy.ContextDialer = proxy.Direct
	if proxyAddress != "" {
		d, err := proxy.SOCKS5("tcp", proxyAddress, nil, nil)
		if err != nil {
			log.Fatalf("failed to make dialer for proxy server at %q: %v", proxyAddress, err)
		}
		dialer = d.(proxy.ContextDialer)
	}
	return dialer
}
