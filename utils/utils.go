package utils

import (
	"os"
	"strings"
	"time"
)

func ReplaceFile(filename, text string) error {
	// Open the file in append mode with standard permissions (0666 should cause UMAKS to be applied)
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the string to the file
	_, err = file.WriteString(text)
	if err != nil {
		return err
	}

	return nil
}

func AppendToFile(filename, text string) error {
	// Open the file in append mode with standard permissions (0666 should cause UMAKS to be applied)
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the string to the file
	_, err = file.WriteString(text)
	if err != nil {
		return err
	}

	return nil
}

func WaitForPrompt(output *ThreadSafeBuffer, timeLimit time.Duration) {
	detectPrompt := make(chan bool)
	go func() {
		start := time.Now()
		for {
			ostr := output.String()
			if strings.Contains(ostr, "#") || strings.Contains(ostr, ">") || strings.Contains(ostr, "$") {
				close(detectPrompt)
				break
			}
			time.Sleep(20 * time.Millisecond)

			// Make sure to stop after 2 seconds.
			if time.Since(start) > timeLimit {
				break
			}
		}
	}()
	select {
	case <-time.After(timeLimit):
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
			if time.Since(start) > timeLimit {
				break
			}
		}
	}()
	select {
	case <-time.After(timeLimit):
	case <-detectPrompt:
	}
}

func Dos2Unix(s string) string {
  return strings.ReplaceAll(s, "\r", "\n")
}
