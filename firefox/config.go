package firefox

import (
	"fmt"
	"os"
	"path/filepath"

	"git.sp4ke.xyz/sp4ke/gomark/browsers"
	"git.sp4ke.xyz/sp4ke/gomark/config"
	"git.sp4ke.xyz/sp4ke/gomark/database"
	"git.sp4ke.xyz/sp4ke/gomark/mozilla"
	"git.sp4ke.xyz/sp4ke/gomark/parsing"
	"git.sp4ke.xyz/sp4ke/gomark/tree"
	"git.sp4ke.xyz/sp4ke/gomark/utils"

	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"
)

const (
	BrowserName = "firefox"
	configDir   = "$HOME/.mozilla/firefox"
)

var (

	// user mutable config
	FFConfig = &FirefoxConfig{
		BrowserConfig: &browsers.BrowserConfig{
			Name:           BrowserName,
			Type:           browsers.TFirefox,
			BkDir:          "",
			BkFile:         "",
			WatchedPaths:   []string{},
            //TODO: Initialize BufferDB
            //TODO: initialize URLIndex
			NodeTree:       &tree.Node{
                Name: "root",
                Parent: nil,
                Type: tree.RootNode,
            },
			Stats:          &parsing.Stats{},
			UseFileWatcher: true,
		},

		// Default data source name query options for `places.sqlite` db
		PlacesDSN: database.DsnOptions{
			"_journal_mode": "WAL",
		},

		// default profile to use
		DefaultProfile: "default",

		WatchAllProfiles: false,
	}

	firefoxProfile = &mozilla.INIProfileLoader{
		//BasePath to be set at runtime in init
		ProfilesFile: mozilla.ProfilesFile,
	}

	FirefoxProfileManager = &mozilla.MozProfileManager{
		BrowserName: BrowserName,
		ConfigDir:   configDir,
		PathGetter:  firefoxProfile,
	}
)

func init() {
	config.RegisterConfigurator(BrowserName, FFConfig)
	config.RegisterConfReadyHooks(initFirefoxConfig)
}

// Schema for config parameters to pass on to firefox that can be overriden by
// users. Options defined here will automatically supported in the
// config.toml file as well as the command line flags.
// New command line flags or config file options will only be accepted if they
// are defined here.
type FirefoxConfig struct {
	// Default data source name query options for `places.sqlite` db
	PlacesDSN        database.DsnOptions
	WatchAllProfiles bool
	DefaultProfile   string

	// Embed base browser config
	*browsers.BrowserConfig
}

func (fc *FirefoxConfig) Set(opt string, v interface{}) error {
	//log.Debugf("setting option %s = %v", opt, v)
	s := structs.New(fc)
	f, ok := s.FieldOk(opt)
	if !ok {
		return fmt.Errorf("%s option not defined", opt)
	}

	return f.Set(v)
}

func (fc *FirefoxConfig) Get(opt string) (interface{}, error) {
	s := structs.New(fc)
	f, ok := s.FieldOk(opt)
	if !ok {
		return nil, fmt.Errorf("%s option not defined", opt)
	}

	return f.Value(), nil
}

func (fc *FirefoxConfig) Dump() map[string]interface{} {
	s := structs.New(fc)
	return s.Map()
}

func (fc *FirefoxConfig) String() string {
	s := structs.New(fc)
	return fmt.Sprintf("%v", s.Map())
}

func (fc *FirefoxConfig) MapFrom(src interface{}) error {
	return mapstructure.Decode(src, fc)
}

func initFirefoxConfig() {
	log.Debugf("<firefox> initializing config")

	// expand env variables to config dir
	pm := FirefoxProfileManager
	pm.ConfigDir = filepath.Join(os.ExpandEnv(configDir))

	// Check if base folder exists
	configFolderExists, err := utils.CheckDirExists(pm.ConfigDir)
	if !configFolderExists {
		log.Criticalf("the base firefox folder <%s> does not exist", pm.ConfigDir)
	}

	if err != nil {
		log.Critical(err)
	}

	firefoxProfile.BasePath = pm.ConfigDir

	//_TODO: allow override with flag --firefox-profile-dir (rename pref default-profile)

	// set global firefox bookmark dir
	//FIX: bookmarkDir is used in created instance of FF before it is setup in config
	bookmarkDir, err := FirefoxProfileManager.GetDefaultProfilePath()
	if err != nil {
		log.Fatal(err)
	}

	// update bookmark dir in firefox config
    //TEST: verify that bookmark dir is set before browser is started
	FFConfig.BkDir = bookmarkDir

	log.Debugf("Using default profile %s", bookmarkDir)
}
