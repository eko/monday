package helper

import (
	"fmt"
	"os"
)

// CheckPathExists checks if application path exists
func CheckPathExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("unable to find application path: %s", path)
	}

	return nil
}
