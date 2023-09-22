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

const startTclSh = "tclsh"
const configTemplateOpen = "puts [open \"flash:configlet\" w+] {"
const configTemplateClose = "}"
const quitTclSh = "exit"
const commitConfig = "copy flash:configlet running-config"
const confirm = "y"

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

// Push pushes a configlet to an ios device.
func Push(opts *Options, device string, username string, password string, configlet string) (string, error) {
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

	tcpconn, err := opts.dialer("tcp", addr)
	if err != nil {
		return "", fmt.Errorf("failed to connect to device %q as user %q: %v", device, username, err)
	}
	sshconn, chans, reqs, err := ssh.NewClientConn(tcpconn, addr, config)
	if err != nil {
		return "", fmt.Errorf("failed to connect to device %q as user %q: %v", device, username, err)
	}
	conn := ssh.NewClient(sshconn, chans, reqs)
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

	if _, err := stdinBuf.Write([]byte(noMore + "\r\n")); err != nil {
		return "", fmt.Errorf("failed to run command %q on device %q: %v", noMore, device, err)
  }
	utils.WaitForPrompt(&output, 2*time.Second)

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
  time.Sleep(200 * time.Millisecond)
	if _, err := stdinBuf.Write([]byte(quitTclSh + "\r\n")); err != nil {
		return "", fmt.Errorf("failed to run command %q on device %q: %v", quitTclSh, device, err)
  }
  time.Sleep(200 * time.Millisecond)
	if _, err := stdinBuf.Write([]byte(commitConfig + "\r\n")); err != nil {
		return "", fmt.Errorf("failed to run command %q on device %q: %v", commitConfig, device, err)
  }
	if _, err := stdinBuf.Write([]byte(confirm + "\r\n")); err != nil {
		return "", fmt.Errorf("failed to run command %q on device %q: %v", confirm, device, err)
  }
  time.Sleep(200 * time.Millisecond)
	if _, err := stdinBuf.Write([]byte(exitCommand + "\r\n")); err != nil {
		return "", fmt.Errorf("failed to run command %q on device %q: %v", exitCommand, device, err)
  }
  time.Sleep(200 * time.Millisecond)

	utils.WaitForPrompt(&output, 2*time.Second)

  return output.String(), nil
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

	tcpconn, err := opts.dialer("tcp", addr)
	if err != nil {
		return "", fmt.Errorf("failed to connect to device %q as user %q: %v", device, username, err)
	}
	sshconn, chans, reqs, err := ssh.NewClientConn(tcpconn, addr, config)
	if err != nil {
		return "", fmt.Errorf("failed to connect to device %q as user %q: %v", device, username, err)
	}
	conn := ssh.NewClient(sshconn, chans, reqs)
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
	if !strings.HasSuffix(cmd, "\r\n") {
		cmd = cmd + "\r\n"
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
