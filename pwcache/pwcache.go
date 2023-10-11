package pwcache

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/user"
	"syscall"

	"golang.org/x/term"
)

// GetPassword gets the password, or reads the cached password from /dev/shm.
func GetPassword(clearCache bool, usePwCache bool) (string, error) {
	userName, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("unable to get current username: %w", err)
	}
	fn := fmt.Sprintf("/dev/shm/gpcache-%s", userName.Username)

	if clearCache {
		err := os.Remove(fn)
		if err != nil {
			return "", fmt.Errorf("failed to delete password cache in %q: %w", fn, err)
		}
	}

	var cachedPw []byte
	if usePwCache {
		cachedPw, err := os.ReadFile(fn)
		if err == nil {
			pw, err := base64.StdEncoding.DecodeString(string(cachedPw))
			if err == nil {
				return string(pw), nil
			}
		}
	}

	fmt.Print("Please enter password: ")
	bytePassword, err := term.ReadPassword(syscall.Stdin)
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	password := string(bytePassword)

	if usePwCache {
		cachedPw = []byte(base64.StdEncoding.EncodeToString([]byte(password)))
		err = os.WriteFile(fn, cachedPw, 0600)
		if err != nil {
			// Non-fatal error.
			log.Printf("failed to cache password in %q: %v", fn, err)
		}
	}

	return password, nil
}
