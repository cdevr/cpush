// Package pwcache allows password entry and saves the password entered in a secure user-only directory (memory backed, so it's gone at reboot)
package pwcache

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"syscall"

	"golang.org/x/term"
)

// getCacheDir returns a secure directory for password cache, creating it with proper permissions if needed.
func getCacheDir() (string, error) {
	userName, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("unable to get current username: %w", err)
	}

	// Use /dev/shm if available (Linux), otherwise fall back to temp dir
	var baseDir string
	if _, err := os.Stat("/dev/shm"); err == nil {
		baseDir = "/dev/shm"
	} else {
		baseDir = os.TempDir()
	}

	// Create user-specific directory with 0700 permissions
	cacheDir := filepath.Join(baseDir, fmt.Sprintf("cpush-cache-%s", userName.Username))
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Verify directory has correct permissions
	info, err := os.Stat(cacheDir)
	if err != nil {
		return "", fmt.Errorf("failed to stat cache directory: %w", err)
	}
	if info.Mode().Perm() != 0700 {
		if err := os.Chmod(cacheDir, 0700); err != nil {
			return "", fmt.Errorf("failed to set cache directory permissions: %w", err)
		}
	}

	return cacheDir, nil
}

// deriveKey derives an encryption key from the username and hostname.
// This is not cryptographically perfect but prevents casual reading of the cache.
func deriveKey() ([]byte, error) {
	userName, err := user.Current()
	if err != nil {
		return nil, err
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	// Derive a key from username and hostname
	material := fmt.Sprintf("%s@%s", userName.Username, hostname)
	hash := sha256.Sum256([]byte(material))
	return hash[:], nil
}

// encrypt encrypts plaintext using AES-GCM.
func encrypt(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decrypt decrypts ciphertext using AES-GCM.
func decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// GetPassword gets the password or reads the cached password from secure storage.
func GetPassword(clearCache bool, usePwCache bool) (string, error) {
	cacheDir, err := getCacheDir()
	if err != nil {
		return "", err
	}
	fn := filepath.Join(cacheDir, "password")

	if clearCache {
		err := os.Remove(fn)
		if err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to delete password cache in %q: %w", fn, err)
		}
		return "", nil
	}

	if usePwCache {
		cachedData, err := os.ReadFile(fn)
		if err == nil {
			key, err := deriveKey()
			if err == nil {
				encryptedData, err := base64.StdEncoding.DecodeString(string(cachedData))
				if err == nil {
					decryptedData, err := decrypt(encryptedData, key)
					if err == nil {
						return string(decryptedData), nil
					}
				}
			}
			// If any decryption step fails, we'll just prompt for password again
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
		key, err := deriveKey()
		if err != nil {
			log.Printf("failed to derive encryption key: %v", err)
		} else {
			encrypted, err := encrypt([]byte(password), key)
			if err != nil {
				log.Printf("failed to encrypt password: %v", err)
			} else {
				encodedData := base64.StdEncoding.EncodeToString(encrypted)
				err = os.WriteFile(fn, []byte(encodedData), 0600)
				if err != nil {
					// Non-fatal error.
					log.Printf("failed to cache password in %q: %v", fn, err)
				}
			}
		}
	}

	return password, nil
}
