package cisco

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cdevr/cpush/options"
	"github.com/cdevr/cpush/utils"
	"golang.org/x/crypto/ssh"
)

const noMore = "terminal length 0" // Command to disable "more" prompt on cisco routers.
const exitCommand = "exit"
const wrCommand = "wr"

const startTclSh = "tclsh"
const configTemplateOpen = "puts [open \"flash:configlet\" w+] {"
const configTemplateClose = "}"
const quitTclSh = "exit"
const commitConfig = "copy flash:configlet running-config"
const confirm = "y"

func isRN(r rune) bool {
	return r == '\r' || r == '\n'
}

func RemovePromptSuffix(str string) string {
	lines := strings.FieldsFunc(str, isRN)
	if len(lines) == 0 {
		return str
	}

	last := len(lines)
	trim := strings.Trim(lines[last-1], " ")
	if strings.HasSuffix(trim, "#") || strings.HasSuffix(trim, ">") {
		last -= 1
	}
	trim = strings.Trim(lines[last-1], " ")
	if strings.HasSuffix(trim, "#"+exitCommand) || strings.HasSuffix(trim, ">"+exitCommand) {
		last -= 1
	}
	return strings.Join(lines[:last], "\n")
}

func respondInteractive(password string) func(user, instruction string, questions []string, echos []bool) ([]string, error) {
	return func(user, instruction string, questions []string, echos []bool) ([]string, error) {
		var answers []string
		for range questions {
			answers = append(answers, password)
		}
		return answers, nil
	}
}

// Push pushes a configlet to an ios device.
func Push(opts *options.Options, device string, username string, password string, configlet string, timeout time.Duration) (string, error) {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
			ssh.KeyboardInteractive(respondInteractive(password)),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Config:          sshConfig(),
	}

	addr := device
	if !strings.Contains(addr, ":") {
		addr = addr + ":22"
	}

	ctx, _ := context.WithTimeout(context.Background(), opts.Timeout)
	tcpConn, err := opts.Dial(ctx, "tcp", addr)
	if err != nil {
		return "", fmt.Errorf("failed to connect to device %q as user %q: %v", device, username, err)
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(tcpConn, addr, config)
	if err != nil {
		return "", fmt.Errorf("failed to connect to device %q as user %q: %v", device, username, err)
	}
	conn := ssh.NewClient(sshConn, chans, reqs)
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

	utils.WaitForPrompt(&output, 2*time.Second, opts.SuppressBanner)

	if _, err := stdinBuf.Write([]byte(noMore + "\r\n")); err != nil {
		return "", fmt.Errorf("failed to run command %q on device %q: %v", noMore, device, err)
	}
	utils.WaitForPrompt(&output, 2*time.Second, false)
	if opts.SuppressSending {
		output.DiscardUntil('\r')
	}

	if _, err := stdinBuf.Write([]byte(startTclSh + "\r\n")); err != nil {
		return "", fmt.Errorf("failed to run command %q on device %q: %v", startTclSh, device, err)
	}
	time.Sleep(200 * time.Millisecond)
	if _, err := stdinBuf.Write([]byte(configTemplateOpen + "\r")); err != nil {
		return "", fmt.Errorf("failed to run command %q on device %q: %v", configTemplateOpen, device, err)
	}
	time.Sleep(200 * time.Millisecond)
	// Expand ";" to \n to allow for multiline.
	expandedConfiglet := strings.ReplaceAll(configlet, ";", "\n")
	for _, s := range strings.Split(expandedConfiglet, "\n") {
		if _, err := stdinBuf.Write([]byte(s + "\r")); err != nil {
			return "", fmt.Errorf("failed to enter configlet line %q on device %q: %v", s, device, err)
		}
		time.Sleep(20)
	}
	if _, err := stdinBuf.Write([]byte(configTemplateClose + "\r")); err != nil {
		return "", fmt.Errorf("failed to run command %q on device %q: %v", configTemplateClose, device, err)
	}
	time.Sleep(100 * time.Millisecond)
	if _, err := stdinBuf.Write([]byte(quitTclSh + "\r")); err != nil {
		return "", fmt.Errorf("failed to run command %q on device %q: %v", quitTclSh, device, err)
	}
	time.Sleep(100 * time.Millisecond)
	if _, err := stdinBuf.Write([]byte(commitConfig + "\r")); err != nil {
		return "", fmt.Errorf("failed to run command %q on device %q: %v", commitConfig, device, err)
	}
	time.Sleep(100 * time.Millisecond)
	utils.WaitFor(&output, "?", 5*time.Second)
	if strings.Contains(output.LastLine(), "Destination filename") {
		if _, err := stdinBuf.Write([]byte("\r")); err != nil {
			return "", fmt.Errorf("failed to confirm write on device %q: %v", device, err)
		}
	} else {
		if _, err := stdinBuf.Write([]byte(confirm + "\r")); err != nil {
			return "", fmt.Errorf("failed to run command %q on device %q: %v", confirm, device, err)
		}
	}
	time.Sleep(200 * time.Millisecond)
	if _, err := stdinBuf.Write([]byte(wrCommand + "\r")); err != nil {
		return "", fmt.Errorf("failed to run command %q on device %q: %v", wrCommand, device, err)
	}
	time.Sleep(200 * time.Millisecond)
	if _, err := stdinBuf.Write([]byte(exitCommand + "\r")); err != nil {
		return "", fmt.Errorf("failed to run command %q on device %q: %v", exitCommand, device, err)
	}
	time.Sleep(200 * time.Millisecond)

	utils.WaitForPrompt(&output, 2*time.Second, false)

	if strings.Contains(output.String(), "Invalid input") {
		var errorLines []string
		for _, line := range strings.Split(output.String(), "\n") {
			if strings.Contains(line, "Invalid input") {
				errorLines = append(errorLines, line)
			}
		}

		return output.String(), fmt.Errorf("error in configlet: %s\n%s", strings.Join(errorLines, "\n"), output.String())
	}

	return output.String(), nil
}

// sshConfig returns additional ssh configuration options for cisco routers, such as allowing bad ciphers used by Cisco.
func sshConfig() ssh.Config {
	extraCiphers := []string{"aes128-cbc", "3des-cbc", "aes192-cbc", "aes256-cbc"}

	config := ssh.Config{}
	config.SetDefaults()

	config.Ciphers = append(config.Ciphers, extraCiphers...)

	return config
}

// Cmd executes a command on a device and returns the output.
func Cmd(opts *options.Options, device string, username string, password string, cmd string, timeout time.Duration) (string, error) {
	log.Printf("cmd called with timeout of %v", timeout)
	// Sets the time when the timeout has elapsed.
	var whenToQuit = time.Now().Add(timeout)

	// if time.Now().After(whenToQuit) {
	// 	return "", fmt.Errorf("timeout executing command on %q: %v elapsed", device, timeout)
	// }

	var result bytes.Buffer

	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
			ssh.KeyboardInteractive(respondInteractive(password)),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Config:          sshConfig(),
	}

	addr := device
	if !strings.Contains(addr, ":") {
		addr = addr + ":22"
	}

	ctx, _ := context.WithTimeout(context.Background(), opts.Timeout)
	tcpConn, err := opts.Dial(ctx, "tcp", addr)
	if err != nil {
		return "", fmt.Errorf("failed to connect to device %q as user %q: %v", device, username, err)
	}
	defer tcpConn.Close()

	if time.Now().After(whenToQuit) {
		return "", fmt.Errorf("timeout executing command on %q: %v elapsed", device, timeout)
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(tcpConn, addr, config)
	if err != nil {
		return "", fmt.Errorf("failed to connect to device %q as user %q: %v", device, username, err)
	}
	conn := ssh.NewClient(sshConn, chans, reqs)
	defer conn.Close()

	if time.Now().After(whenToQuit) {
		return "", fmt.Errorf("timeout executing command on %q: %v elapsed", device, timeout)
	}

	session, err := conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to get session on device %q: %v", device, err)
	}
	defer session.Close()

	if time.Now().After(whenToQuit) {
		return "", fmt.Errorf("timeout executing command on %q: %v elapsed", device, timeout)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO: 0,
	}

	if err := session.RequestPty("xterm", 50, 80, modes); err != nil {
		return "", fmt.Errorf("failed to get pty on device %q: %v", device, err)
	}

	if time.Now().After(whenToQuit) {
		return "", fmt.Errorf("timeout executing command on %q: %v elapsed", device, timeout)
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

	utils.WaitForPrompt(&output, 2*time.Second, opts.SuppressBanner)

	if time.Now().After(whenToQuit) {
		return "", fmt.Errorf("timeout executing command on %q: %v elapsed", device, timeout)
	}

	if !opts.SuppressSending {
		fmt.Fprintf(&result, "sending %q", noMore)
	}
	if _, err := stdinBuf.Write([]byte(noMore + "\r")); err != nil {
		return "", fmt.Errorf("failed to run command %q on device %q: %v", noMore, device, err)
	}
	if opts.SuppressAdmin {
		utils.WaitForPrompt(&output, 2*time.Second, false)
		output.Reset()
	}

	if time.Now().After(whenToQuit) {
		return "", fmt.Errorf("timeout executing command on %q: %v elapsed", device, timeout)
	}

	if !opts.SuppressSending {
		fmt.Fprintf(&result, "sending %q", cmd)
	}
	if !strings.HasSuffix(cmd, "\r") {
		cmd = cmd + "\r"
	}
	if _, err := stdinBuf.Write([]byte(cmd)); err != nil {
		return "", fmt.Errorf("failed to run command %q on device %q: %v", cmd, device, err)
	}
	if opts.SuppressSending {
		utils.WaitForEnter(&output, 2*time.Second)
		output.DiscardUntil('\r')
	}
	time.Sleep(200 * time.Millisecond)

	if time.Now().After(whenToQuit) {
		return "", fmt.Errorf("timeout executing command on %q: %v elapsed", device, timeout)
	}

	if !opts.SuppressSending {
		fmt.Fprintf(&result, "sending %q", exitCommand)
	}
	if _, err := stdinBuf.Write([]byte(exitCommand + "\r\n")); err != nil {
		return "", fmt.Errorf("failed to run command %q on device %q: %v", cmd, device, err)
	}

	done := make(chan struct{})
	go func() {
		session.Wait()
		fmt.Fprintf(&result, "%s", output.String())

		close(done)
	}()

	remainingTime := time.Until(whenToQuit)
	select {
	case <-done:
		// Determine the
	case <-time.After(remainingTime):
		return "", fmt.Errorf("timeout of %v hit", opts.Timeout)
	}
	return RemovePromptSuffix(result.String()), nil
}
