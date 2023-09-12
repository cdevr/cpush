package utils

import "os"

func AppendToFile(filename, text string) error {
	// Open the file in append mode with write-only permissions
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
