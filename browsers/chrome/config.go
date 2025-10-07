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

package chrome

import (
	"github.com/blob42/gosuki/pkg/config"
	"github.com/blob42/gosuki/pkg/modules"
	"github.com/blob42/gosuki/pkg/profiles"
	"github.com/blob42/gosuki/pkg/tree"
)

const (
	BrowserName    = "chrome" // Default flavour to use
	DefaultProfile = "Default"
	RootNodeName   = "ROOT"
)

type ChromeConfig struct {
	*modules.BrowserConfig `toml:"-"`
	modules.ProfilePrefs   `toml:"profile-options" mapstructure:"profile-options"`
	CustomProfiles         []profiles.CustomProfile `toml:"custom-profiles" mapstructure:"custom-profiles"`
}

var (
	ProfileManager = &ChromeProfileManager{}

	ChromeCfg = NewChromeConfig()
)

func NewChromeConfig() *ChromeConfig {

	config := &ChromeConfig{
		BrowserConfig: &modules.BrowserConfig{
			Name:   BrowserName,
			BkFile: "Bookmarks",
			NodeTree: &tree.Node{
				Title:  RootNodeName,
				Parent: nil,
				Type:   tree.RootNode,
			},
			UseFileWatcher: true,
			UseHooks:       []string{"node_tags_from_name"},
		},
		ProfilePrefs: modules.ProfilePrefs{
			Profile:          DefaultProfile,
			WatchAllProfiles: true,
		},
	}

	return config
}

func init() {
	config.RegisterConfigurator(BrowserName, config.AsConfigurator(ChromeCfg))
}
