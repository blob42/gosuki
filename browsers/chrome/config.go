package chrome

import (
	"git.blob42.xyz/gomark/gosuki/pkg/modules"
	"git.blob42.xyz/gomark/gosuki/pkg/parsing"
	"git.blob42.xyz/gomark/gosuki/pkg/tree"
)

const (
	BrowserName    = "chrome"
	ChromeBaseDir  = "$HOME/.config/google-chrome"
	DefaultProfile = "Default"
	RootNodeName   = "ROOT"
)

type ChromeConfig struct {
	Profile                string
	*modules.BrowserConfig `toml:"-"`
	modules.ProfilePrefs   `toml:"profile_options"`
}

var (
	ChromeCfg = &ChromeConfig{
		Profile: DefaultProfile,
		BrowserConfig: &modules.BrowserConfig{
			Name:   BrowserName,
			Type:   modules.TChrome,
			BkDir:  "$HOME/.config/google-chrome/Default",
			BkFile: "Bookmarks",
			NodeTree: &tree.Node{
				Name:   RootNodeName,
				Parent: nil,
				Type:   tree.RootNode,
			},
			Stats:          &parsing.Stats{},
			UseFileWatcher: true,
			UseHooks:       []string{"tags_from_name", "notify-send"},
		},
		//TODO: profile
	}
)