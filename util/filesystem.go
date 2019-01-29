package util

import (
	"fmt"
	"os"
)

// Exists checks whether the file exists
func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// CreateDirectoryIfNotExists creates directory if it doesn't yet exist
func CreateDirectoryIfNotExists(path string, mode os.FileMode) error {
	if !Exists(path) {
		if err := os.MkdirAll(path, mode); err != nil {
			return fmt.Errorf("failed to create directory %s: %s", path, err)
		}
	}

	return nil
}
