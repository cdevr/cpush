package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
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

var device = flag.String("device", "", "a device, or list of devices to execute commands on")
var command = flag.String("cmd", "", "a command to execute")

var suppressBanner = flag.Bool("suppress_banner", true, "suppress the SSH banner and login")
var suppressAdmin = flag.Bool("suppress_admin", true, "suppress administrative information")
var suppressSending = flag.Bool("suppress_sending", true, "suppress what is being sent to the router")

var username = flag.String("username", "", "username to use for login")

var timeout = flag.Duration("timeout", 10*time.Second, "timeout for the command")

var cacheAllowed = flag.Bool("pw_cache_allowed", true, "allowed to cache password in /dev/shm")
var clearPwCache = flag.Bool("pw_clear_cache", false, "forcibly clear the pw cache")

const noMore = "terminal length 0" // Command to disable "more" prompt on cisco routers
const exitCommand = "exit"         // Command to disable "more" prompt on cisco routers

// getPassword gets the password, or reads the cached password from /dev/shm.
func getPassword(cacheAllowed, clearCache bool) (string, error) {
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

func main() {
	flag.Parse()

	if *device == "" {
		log.Printf("you didn't pass in a device")
		return
	}
	if *command == "" {
		log.Printf("you didn't pass in a device")
		return
	}
	if *username == "" {
		*username = GetUser()
	}

	password, err := getPassword(*cacheAllowed, *clearPwCache)
	if err != nil {
		log.Fatalf("error getting password for user: %v")
	}

	config := &ssh.ClientConfig{
		User: *username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := *device
	if !strings.Contains(addr, ":") {
		addr = addr + ":22"
	}
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		log.Fatalf("failed to connect to router %q: %v", *device, err)
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		log.Fatalf("failed to get session on router %q: %v", *device, err)
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO: 0,
	}

	if err := session.RequestPty("xterm", 50, 80, modes); err != nil {
		log.Printf("failed to get pty on router %q: %v", *device, err)
		return
	}

	stdinBuf, err := session.StdinPipe()
	if err != nil {
		log.Printf("failed to get stdin connected to remote host %q: %v", *device, err)
		return
	}

	if err := session.Shell(); err != nil {
		log.Printf("failed to get shell on router %q: %v", *device, err)
		return
	}

	var output ThreadSafeBuffer
	session.Stdout = &output

	WaitForPrompt(&output, 2*time.Second)
	if *suppressBanner {
		output.Reset()
	}

	if !*suppressSending {
		log.Printf("sending %q", noMore)
	}
	if _, err := stdinBuf.Write([]byte(noMore + "\r\n")); err != nil {
		log.Printf("failed to run command %q on router %q: %v", noMore, *device, err)
		return
	}
	if *suppressAdmin {
		WaitForPrompt(&output, 2*time.Second)
		output.Reset()
	}

	if !*suppressSending {
		log.Printf("sending %q", *command)
	}
	if !strings.HasSuffix(*command, "\n") {
		*command = *command + "\n"
	}
	if _, err := stdinBuf.Write([]byte(*command)); err != nil {
		log.Printf("failed to run command %q on router %q: %v", *command, *device, err)
		return
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
		log.Printf("sending %q", exitCommand)
	}
	if _, err := stdinBuf.Write([]byte(exitCommand + "\r\n")); err != nil {
		log.Printf("failed to run command %q on router %q: %v", *command, *device, err)
		return
	}

	done := make(chan struct{})
	go func() {
		session.Wait()
		if toPrint != "" {
			fmt.Println(toPrint)
		} else {
			fmt.Println(output.String())
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

	}
}
