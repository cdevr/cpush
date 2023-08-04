package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
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

var username = flag.String("username", "", "username to use for login")

var timeout = flag.Duration("timeout", 10*time.Second, "timeout for the command")

var cacheAllowed = flag.Bool("pw_cache_allowed", true, "allowed to cache password in /dev/shm")
var clearPwCache = flag.Bool("pw_clear_cache", false, "forcibly clear the pw cache")

const noMore = "terminal length 0" // Command to disable "more" prompt on cisco routers
const exitCommand = "exit"         // Command to disable "more" prompt on cisco routers

// GetPassword gets the password, or reads the cached password from /dev/shm.
func GetPassword(cacheAllowed, clearCache bool) (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("Unable to get current username: %w", err)
	}
	fn := fmt.Sprintf("/dev/shm/gpcache-%s", user.Username)

	if clearCache {
		err := os.Remove(fn)
		if err != nil {
			return "", fmt.Errorf("failed to delete password cache in %q: %w", fn, err)
		}
	}

	cachedPw, err := ioutil.ReadFile(fn)
	if err == nil {
		pw, err := base64.StdEncoding.DecodeString(string(cachedPw))
		if err == nil {
			return string(pw), nil
		}
	}

	fmt.Print("Please enter the password to use: ")
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	password := string(bytePassword)

	cachedPw = []byte(base64.StdEncoding.EncodeToString([]byte(password)))
	err = ioutil.WriteFile(fn, cachedPw, 0600)
	if err != nil {
		// Non-fatal error.
		log.Printf("failed to cache password in %q: %v", fn, err)
	}

	return password, nil
}

type ThreadSafeBuffer struct {
	b bytes.Buffer
	m sync.Mutex
}

func (b *ThreadSafeBuffer) Read(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Read(p)
}

func (b *ThreadSafeBuffer) Write(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Write(p)
}

func (b *ThreadSafeBuffer) Len() int {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Len()
}

func (b *ThreadSafeBuffer) Reset() {
	b.m.Lock()
	defer b.m.Unlock()
	b.b.Reset()
}

func (b *ThreadSafeBuffer) String() string {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.String()
}

func WaitForPrompt(output *ThreadSafeBuffer, timeLimit time.Duration) {
	detectPrompt := make(chan bool)
	go func() {
		start := time.Now()
		for {
			if strings.Contains(output.String(), "#") || strings.Contains(output.String(), ">") {
				close(detectPrompt)
				break
			}
			time.Sleep(20 * time.Millisecond)

			// Make sure to stop after 2 seconds.
			if time.Since(start).Seconds() > 2 {
				break
			}
		}
	}()
	select {
	case <-time.After(2 * time.Second):
	case <-detectPrompt:
	}
}

func WaitForEnter(output *ThreadSafeBuffer, timeLimit time.Duration) {
	detectPrompt := make(chan bool)
	go func() {
		start := time.Now()
		for {
			if strings.Contains(output.String(), "\n") {
				close(detectPrompt)
				break
			}
			time.Sleep(20 * time.Millisecond)

			// Make sure to stop after 2 seconds.
			if time.Since(start).Seconds() > 2 {
				break
			}
		}
	}()
	select {
	case <-time.After(2 * time.Second):
	case <-detectPrompt:
	}
}

func RemovePromptSuffix(str string) string {
	lines := strings.Split(str, "\n")
	if len(lines) == 0 {
		return str
	}

	last := len(lines)
	trim := strings.Trim(lines[last-1], " ")
	if strings.HasSuffix(trim, "#") || strings.HasSuffix(trim, ">") {
		last -= 1
	}
	return strings.Join(lines[:last], "\n")
}

// Push pushes a configlet to a device.
func Push() {
}

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

func respondInteractive(password string) (func(user, instruction string, questions []string, echos []bool) ([]string, error)) {
	return func(user, instruction string, questions []string, echos []bool) ([]string, error) {
		answers := []string{}
		for range questions {
			answers = append(answers, password)
		}
		return answers, nil
	}
}
// Cmd executes a command on a device and returns the output.
func Cmd(device string, username string, password string, cmd string) (string, error) {
	var result bytes.Buffer

	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
			ssh.KeyboardInteractive(respondInteractive(password)),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := device
	if !strings.Contains(addr, ":") {
		addr = addr + ":22"
	}
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return "", fmt.Errorf("failed to connect to device %q as user %q: %v", device, username, err)
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to get session on device %q: %v", device, err)
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO: 0,
	}

	if err := session.RequestPty("xterm", 50, 80, modes); err != nil {
		return "", fmt.Errorf("failed to get pty on device %q: %v", device, err)
	}

	stdinBuf, err := session.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stdin connected to remote host %q: %v", device, err)
	}

	if err := session.Shell(); err != nil {
		return "", fmt.Errorf("failed to get shell on device %q: %v", device, err)
	}

	var output ThreadSafeBuffer
	session.Stdout = &output

	WaitForPrompt(&output, 2*time.Second)
	if *suppressBanner {
		output.Reset()
	}

	if !*suppressSending {
		fmt.Fprintf(&result, "sending %q", noMore)
	}
	if _, err := stdinBuf.Write([]byte(noMore + "\r\n")); err != nil {
		return "", fmt.Errorf("failed to run command %q on device %q: %v", noMore, device, err)
	}
	if *suppressAdmin {
		WaitForPrompt(&output, 2*time.Second)
		output.Reset()
	}

	if !*suppressSending {
		fmt.Fprintf(&result, "sending %q", *command)
	}
	if !strings.HasSuffix(*command, "\n") {
		*command = *command + "\n"
	}
	if _, err := stdinBuf.Write([]byte(*command)); err != nil {
		return "", fmt.Errorf("failed to run command %q on device %q: %v", *command, device, err)
	}
	time.Sleep(200 * time.Millisecond)
	if *suppressSending {
		WaitForEnter(&output, 2*time.Second)
		output.Reset()
	}

	toPrint := ""
	if *suppressAdmin {
		WaitForPrompt(&output, 2*time.Second)
		toPrint = output.String()
	}

	if !*suppressSending {
		fmt.Fprintf(&result, "sending %q", exitCommand)
	}
	if _, err := stdinBuf.Write([]byte(exitCommand + "\r\n")); err != nil {
		return "", fmt.Errorf("failed to run command %q on device %q: %v", *command, device, err)
	}

	done := make(chan struct{})
	go func() {
		session.Wait()
		if toPrint != "" {
			fmt.Fprintf(&result, "%s", toPrint)
		} else {
			fmt.Fprintf(&result, "%s", output.String())
		}

		close(done)
	}()

	select {
	case <-done:
	case <-time.After(*timeout):
		return "", fmt.Errorf("timeout hit!")
	}
	return RemovePromptSuffix(result.String()), nil
}

// CmdDevices executes a command on many devices, prints the output.

type routerOutput struct {
	router string
	output string
}

func CmdDevices(devices []string, username string, password string, cmd string) {
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
				output, err = Cmd(device, username, password, cmd)
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
	flag.Parse()

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

	password, err := GetPassword(*cacheAllowed, *clearPwCache)
	if err != nil {
		log.Fatalf("error getting password for user: %v", err)
	}
	if *clearPwCache {
		return
	}

	if *device != "" {
		output, err := Cmd(*device, *username, password, *command)
		if err != nil {
			log.Fatalf("failed to execute command %q on device %q: %v", *command, *device, err)
		}
		fmt.Printf("%s\n", output)
	} else if *deviceList != "" {
		devices := strings.Split(*deviceList, ",")

		CmdDevices(devices, *username, password, *command)
	} else if *deviceFile != "" {
		fileLines, err := ioutil.ReadFile(*deviceFile)
		if err != nil {
			log.Fatalf("failed to read device file %q: %v", *deviceFile, err)
		}
		devices := strings.Split(string(fileLines), "\n")

		CmdDevices(devices, *username, password, *command)
	} else if *deviceStdIn {
		fileLines, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("failed to read devices from stdin %q: %v", *deviceFile, err)
		}
		devices := strings.Split(string(fileLines), "\n")

		CmdDevices(devices, *username, password, *command)
	}
	os.Exit(0)
}
