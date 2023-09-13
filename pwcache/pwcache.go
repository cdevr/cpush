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

	cachedPw, err := os.ReadFile(fn)
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
	err = os.WriteFile(fn, cachedPw, 0600)
	if err != nil {
		// Non-fatal error.
		log.Printf("failed to cache password in %q: %v", fn, err)
	}

	return password, nil
}
