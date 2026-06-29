//go:build !windows

package database

import (
	"github.com/blob42/gosuki/internal/utils"
)

// getPlatformDBDir returns the platform-specific directory for the database.
// On non-Windows platforms this uses the XDG data directory (~/.local/share/gosuki).
func getPlatformDBDir() (string, error) {
	return utils.GetGosukiDataDir()
}
