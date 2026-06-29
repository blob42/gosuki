package utils

import (
	"os"
	"path/filepath"
)

// GetGosukiDataDir returns the platform-appropriate gosuki data directory.
// On Windows this uses the same directory as the config file (%APPDATA%\gosuki).
func GetGosukiDataDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, GosukiDirName), nil
}
