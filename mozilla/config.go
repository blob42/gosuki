package mozilla

import (
	"git.sp4ke.xyz/sp4ke/gomark/database"
	"git.sp4ke.xyz/sp4ke/gomark/logging"
)

const (
	BookmarkFile = "places.sqlite"
)

var (
	log          = logging.GetLogger("FF")
	ConfigFolder = "$HOME/.mozilla/firefox"
)

var FirefoxDefaultConfig = &FirefoxConfig{

	// Default data source name query options for `places.sqlite` db
	PlacesDSN: database.DsnOptions{
		"_journal_mode": "WAL",
	},

	// default profile to use
	DefaultProfile: "default",

	WatchAllProfiles: false,
}
