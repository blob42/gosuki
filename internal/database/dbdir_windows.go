package database

import (
	"github.com/blob42/gosuki/internal/utils"
)

// getPlatformDBDir returns the platform-specific directory for the database.
// On Windows this uses the same directory as the config file (%APPDATA%\gosuki).
func getPlatformDBDir() (string, error) {
	return utils.GetGosukiDataDir()
}
