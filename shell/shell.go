package shell

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cdevr/cpush/options"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

func respondInteractive(password string) func(user, instruction string, questions []string, echos []bool) ([]string, error) {
	return func(user, instruction string, questions []string, echos []bool) ([]string, error) {
		var answers []string
		for range questions {
			answers = append(answers, password)
		}
		return answers, nil
	}
}

// sshConfig returns additional ssh configuration options for cisco routers, such as allowing bad ciphers used by Cisco.
func sshConfig() ssh.Config {
	extraCiphers := []string{"aes128-cbc", "3des-cbc", "aes192-cbc", "aes256-cbc"}

	config := ssh.Config{}
	config.SetDefaults()

	config.Ciphers = append(config.Ciphers, extraCiphers...)

	return config
}

// Interactive starts a remote shell and connects it to the terminal.
func Interactive(opts *options.Options, device string, username string, password string) error {
	log.Printf("starting interactive shell")
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

	tcpconn, err := opts.Dial(context.Background(), "tcp", addr)
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

	// Set terminal to raw mode so single keys work.
	oldTerminalState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to set Terminal to raw mode: %v", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldTerminalState)

	session.Stdin = os.Stdin
	session.Stderr = os.Stderr
	session.Stdout = os.Stdout

	if err := session.Shell(); err != nil {
		return fmt.Errorf("failed to get shell on device %q: %v", device, err)
	}

	return session.Wait()
}
