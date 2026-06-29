//go:build !windows

package utils

import (
	"path/filepath"
)

// GetGosukiDataDir returns the platform-appropriate gosuki data directory.
// On non-Windows platforms this uses the XDG data directory (~/.local/share/gosuki).
func GetGosukiDataDir() (string, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, GosukiDirName), nil
}
