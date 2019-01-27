package util

import (
	"fmt"
	"os"
)

// FileExists checks whether the file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// CreateDirectoryIfNotExists creates directory if it doesn't yet exist
func CreateDirectoryIfNotExists(path string, mode os.FileMode) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(path, mode); err != nil {
				return fmt.Errorf("failed to create directory %s: %s", path, err)
			}
		} else {
			return fmt.Errorf("failed to stat directory %s: %s", path, err)
		}
	}

	return nil
}
