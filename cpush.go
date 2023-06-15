package main

import (
        "flag"
        "time"
        "fmt"
        "log"
        "syscall"
        "bytes"
        "strings"

        "golang.org/x/crypto/ssh"
        "golang.org/x/term"
)

var device = flag.String("device", "", "a device to execute commands on")
var command = flag.String("cmd", "", "a command to execute")

var username = flag.String("username", "", "username to use for login")

var timeout = flag.Duration("timeout", 10 * time.Second, "timeout for the command")

const noMore = "terminal length 0" // Command to disable "more" prompt on cisco routers
const exitCommand = "exit" // Command to disable "more" prompt on cisco routers

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
                log.Printf("you didn't pass in a device")
                return
        }

        fmt.Print("Please enter the password to use: ")
        bytePassword, err := term.ReadPassword(int(syscall.Stdin))
        fmt.Println()
        if err != nil {
                log.Printf("failed to read password")
                return
        }

        fmt.Print("Please enter the password to use: ")
        bytePassword, err := term.ReadPassword(int(syscall.Stdin))
        fmt.Println()
        if err != nil {
                log.Printf("failed to read password")
                return
        }
        password := string(bytePassword)

        config := &ssh.ClientConfig{
                User: *username,
                Auth: []ssh.AuthMethod{
                        ssh.Password(password),
                },
                HostKeyCallback: ssh.InsecureIgnoreHostKey(),
        }

        addr := *device
        if ! strings.Contains(addr, ":") {
                addr = addr + ":22"
        }
        conn, err := ssh.Dial("tcp", addr, config)
        if err != nil {
                log.Printf("failed to connect to router %q: %v", *device, err)
                return
        }
        defer conn.Close()

        session, err := conn.NewSession()
        if err != nil {
                log.Printf("failed to get session on router %q: %v", *device, err)
                return
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

        var buf bytes.Buffer
        session.Stdout = &buf

        log.Printf("sending %q", noMore)
        if _, err := stdinBuf.Write([]byte(noMore + "\r\n")); err != nil {
                log.Printf("failed to run command %q on router %q: %v", noMore, *device, err)
                return
        }
        time.Sleep(200 * time.Millisecond)
        buf.Reset()

        log.Printf("sending %q", *command)
        if _, err := stdinBuf.Write([]byte(*command + "\r\n")); err != nil {
                log.Printf("failed to run command %q on router %q: %v", *command, *device, err)
                return
        }
        time.Sleep(200 * time.Millisecond)

        log.Printf("sending %q", exitCommand)
        if _, err := stdinBuf.Write([]byte(exitCommand + "\r\n")); err != nil {
                log.Printf("failed to run command %q on router %q: %v", *command, *device, err)
                return
        }

        done := make(chan struct{})
        go func() {
                session.Wait()
                fmt.Println(buf.String())

                close(done)
        }()

        select {
                case <- done:
                case <- time.After(*timeout):
                        log.Printf("timeout hit!")
        }
}
