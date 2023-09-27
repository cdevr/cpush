package shell

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cdevr/cpush/cisco"
	"golang.org/x/crypto/ssh"
)

func respondInteractive(password string) func(user, instruction string, questions []string, echos []bool) ([]string, error) {
	return func(user, instruction string, questions []string, echos []bool) ([]string, error) {
		answers := []string{}
		for range questions {
			answers = append(answers, password)
		}
		return answers, nil
	}
}

// Interactive starts a remote shell and connects it to the terminal.
func Interactive(opts *cisco.Options, device string, username string, password string) error {
	log.Printf("starting interactive shell")
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

	tcpconn, err := opts.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to device %q as user %q: %v", device, username, err)
	}
	sshconn, chans, reqs, err := ssh.NewClientConn(tcpconn, addr, config)
	if err != nil {
		return fmt.Errorf("failed to connect to device %q as user %q: %v", device, username, err)
	}
	conn := ssh.NewClient(sshconn, chans, reqs)
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to get session on device %q: %v", device, err)
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO: 0,
	}

	if err := session.RequestPty("xterm", 50, 80, modes); err != nil {
		return fmt.Errorf("failed to get pty on device %q: %v", device, err)
	}

	session.Stdin = os.Stdin
	session.Stderr = os.Stderr
	session.Stdout = os.Stdout

	if err := session.Shell(); err != nil {
		return fmt.Errorf("failed to get shell on device %q: %v", device, err)
	}

	return session.Wait()
}