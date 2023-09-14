package cisco

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/cdevr/cpush/utils"
	"golang.org/x/crypto/ssh"
)

const noMore = "terminal length 0" // Command to disable "more" prompt on cisco routers.
const exitCommand = "exit"         // Command to disable "more" prompt on cisco routers.

// Push pushes a configlet to an ios device.
func Push() {
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

func respondInteractive(password string) func(user, instruction string, questions []string, echos []bool) ([]string, error) {
	return func(user, instruction string, questions []string, echos []bool) ([]string, error) {
		answers := []string{}
		for range questions {
			answers = append(answers, password)
		}
		return answers, nil
	}
}

// Cmd executes a command on a device and returns the output.
func Cmd(opts *Options, device string, username string, password string, cmd string) (string, error) {
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

	var output utils.ThreadSafeBuffer
	session.Stdout = &output

	utils.WaitForPrompt(&output, 2*time.Second)
	if opts.suppressBanner {
		output.Reset()
	}

	if !opts.suppressSending {
		fmt.Fprintf(&result, "sending %q", noMore)
	}
	if _, err := stdinBuf.Write([]byte(noMore + "\r\n")); err != nil {
		return "", fmt.Errorf("failed to run command %q on device %q: %v", noMore, device, err)
	}
	if opts.suppressAdmin {
		utils.WaitForPrompt(&output, 2*time.Second)
		output.Reset()
	}

	if !opts.suppressSending {
		fmt.Fprintf(&result, "sending %q", cmd)
	}
	if !strings.HasSuffix(cmd, "\n") {
		cmd = cmd + "\n"
	}
	if _, err := stdinBuf.Write([]byte(cmd)); err != nil {
		return "", fmt.Errorf("failed to run command %q on device %q: %v", cmd, device, err)
	}
	time.Sleep(200 * time.Millisecond)
	if opts.suppressSending {
		utils.WaitForEnter(&output, 2*time.Second)
		output.Reset()
	}

	toPrint := ""
	if opts.suppressAdmin {
		utils.WaitForPrompt(&output, 2*time.Second)
		toPrint = output.String()
	}

	if !opts.suppressSending {
		fmt.Fprintf(&result, "sending %q", exitCommand)
	}
	if _, err := stdinBuf.Write([]byte(exitCommand + "\r\n")); err != nil {
		return "", fmt.Errorf("failed to run command %q on device %q: %v", cmd, device, err)
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
	case <-time.After(opts.timeout):
		return "", fmt.Errorf("timeout of %v hit!", opts.timeout)
	}
	return RemovePromptSuffix(result.String()), nil
}
