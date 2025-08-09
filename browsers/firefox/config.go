//
// Copyright (c) 2023-2025 Chakib Ben Ziane <contact@blob42.xyz> and [`GoSuki` contributors]
// (https://github.com/blob42/gosuki/graphs/contributors).
//
// All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This file is part of GoSuki.
//
// GoSuki is free software: you can redistribute it and/or modify it under the terms of
// the GNU Affero General Public License as published by the Free Software Foundation,
// either version 3 of the License, or (at your option) any later version.
//
// GoSuki is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
// without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR
// PURPOSE.  See the GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License along with
// gosuki.  If not, see <http://www.gnu.org/licenses/>.

// Configuration
package firefox

import (
	"github.com/blob42/gosuki/internal/database"
	"github.com/blob42/gosuki/pkg/browsers/mozilla"
	"github.com/blob42/gosuki/pkg/config"
	"github.com/blob42/gosuki/pkg/modules"
	"github.com/blob42/gosuki/pkg/profiles"
	"github.com/blob42/gosuki/pkg/tree"
)

const (
	DefaultProfile = "default"

	// Default flavour to use
	BrowserName = "firefox"
)

var (

	// firefox global config state.
	FFConfig *FirefoxConfig

	ffProfileLoader = &profiles.INIProfileLoader{
		//BasePath to be set at runtime in init
		ProfilesFile: mozilla.ProfilesFile,
	}

	FirefoxProfileManager = mozilla.NewMozProfileManager(
		ffProfileLoader,
	)
)

// FirefoxConfig implements the Configurator interface
// which allows it to register and set field through the Configurator.
//
// It is also used alongside cmd_flags.go to dynamically register cli flags
// that can change this config (struct fields) from command line at runtime.
//
// The struct schema defines the parameters to pass on to firefox that can be
// overriden by users. Options defined here will automatically supported in the
// config.toml file as well as the command line flags. New command line flags or
// config file options will only be accepted if they are defined here.
type FirefoxConfig struct {
	// Default data source name query options for `places.sqlite` db
	PlacesDSN database.DsnOptions `toml:"-"`

	modules.ProfilePrefs `toml:"profile-options" mapstructure:"profile-options"`

	CustomProfiles []profiles.CustomProfile `toml:"custom-profiles" mapstructure:"custom-profiles"`

	//TEST: ignore this field in config.Configurator interface
	// Embed base browser config
	*modules.BrowserConfig `toml:"-"`
}

func NewFirefoxConfig() *FirefoxConfig {

	cfg := &FirefoxConfig{
		BrowserConfig: &modules.BrowserConfig{
			Name:   BrowserName,
			BkFile: mozilla.PlacesFile,
			NodeTree: &tree.Node{
				Title:  mozilla.RootName,
				Parent: nil,
				Type:   tree.RootNode,
			},
			UseFileWatcher: true,
			// NOTE: see parsing.Hook to add custom parsing logic for each
			// parsed bookmark node
			// UseHooks: []string{"node_notify_send"},
			UseHooks: []string{"node_tags_from_name", "node_marktab"},
		},

		// Default data source name query options for `places.sqlite` db
		PlacesDSN: database.DsnOptions{
			"_journal_mode": "WAL",
		},

		// default profile to use, can be selected as cli argument

		ProfilePrefs: modules.ProfilePrefs{
			Profile:          DefaultProfile,
			WatchAllProfiles: true,
		},
	}

	return cfg
}

func init() {
	FFConfig = NewFirefoxConfig()
	config.RegisterConfigurator(BrowserName, config.AsConfigurator(FFConfig))

	// An example of running custom code when config is ready
	// config.RegisterConfReadyHooks(func(c *cli.Context) error{
	// 	// log.Debugf("%#v", config.GetAll().Dump())
	//
	//
	// 	if userConf := config.GetModule(BrowserName); userConf != nil {
	// 		watchAll, _ := userConf.Get("WatchAllProfiles")
	// 		log.Debugf("WATCH_ALL: %v", watchAll)
	// 	}
	//
	// 	return nil
	// })
}
