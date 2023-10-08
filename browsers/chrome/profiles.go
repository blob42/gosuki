// Copyright (c) 2023 Chakib Ben Ziane <contact@blob42.xyz>  and [`gosuki` contributors](https://github.com/blob42/gosuki/graphs/contributors).
// All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This file is part of GoSuki.
//
// GoSuki is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// GoSuki is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with gosuki.  If not, see <http://www.gnu.org/licenses/>.
package chrome

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/blob42/gosuki/internal/utils"
	"github.com/blob42/gosuki/pkg/profiles"
)

// Chrome state file.
// The state file is a json file containing the last used profile and the list
// of profiles. Equivalent to the profiles.ini file for mozilla browsers.
const StateFile = "Local State"

// Chrome flavour names
const (
	ChromeStable = "chrome"
	ChromeUnstable = "chrome-unstable"
	//TODO:
	// Chromium = "chromium"
)


var (
	ChromeBrowsers = map[string]profiles.BrowserFlavour{
		ChromeStable: {ChromeStable, "~/.config/google-chrome" },
	}
)

// Helper struct to manage chrome profiles
// profiles.ProfileManager is implemented at the browser level
type ChromeProfileManager struct {}


// Returns all profiles for a given flavour
func (*ChromeProfileManager) GetProfiles(flavour string) ([]*profiles.Profile, error) {
	var result []*profiles.Profile

	f, ok := ChromeBrowsers[flavour]

	if !ok {
		return nil, fmt.Errorf("unknown flavour <%s>", flavour)
	}
	statePath, err := utils.ExpandPath(f.BaseDir, StateFile)
	if err != nil {
		return nil, err
	}

	state, err := loadLocalState(statePath)
	if err != nil {
		return nil, err
	}

	for id, profile := range state.Profile.InfoCache {
		result = append(result, &profiles.Profile{
			Id:      id,
			Name:    profile.Name,
			Path:    id,
			BaseDir: f.BaseDir,
		})
	}

	return result, nil
}

func (*Chrome) GetProfiles(flavour string) ([]*profiles.Profile, error) {
	return ProfileManager.GetProfiles(flavour)
}

// Returns all flavours supported by this browser
func (*Chrome) ListFlavours() []profiles.BrowserFlavour {
	var result []profiles.BrowserFlavour

	// detect local flavours
	for _, v := range ChromeBrowsers {
		if v.Detect() {
			result = append(result, v)
		}
	}

	return result
}


// If should watch all profiles
func (chrome *Chrome) WatchAllProfiles() bool {
	return chrome.ChromeConfig.WatchAllProfiles
}


// chrome uses ID to identify the profile path
func (cpm *ChromeProfileManager) GetProfileByID (flavour string, id string) (*profiles.Profile, error) {
	profiles, err := cpm.GetProfiles(flavour)
	if err != nil {
		return nil, err
	}

	for _, p := range profiles {
		if p.Id == id {
			return p, nil
		}
	}

	return nil, fmt.Errorf("profile %s not found", id)
}

// Notifies the module to use a custom profile
//NOTE: this is implemented at the browser Level
func (c *Chrome) UseProfile(p profiles.Profile) error {
	c.Profile = p.Name

	// setup the bookmark dir
	if bookmarkDir, err := p.AbsolutePath(); err != nil {
		return err
	} else {
		c.BkDir = bookmarkDir
		return nil
	}
}


type StateData struct {
	LastUsed string
	Profile struct {
		InfoCache map[string]profiles.Profile `json:"info_cache"`
	}
}


func loadLocalState(path string) (*StateData, error) {
	var state StateData
	file, err := os.Open(path)
	if err != nil {
	  return nil, err
	}

	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &state)
	return &state, err
}
