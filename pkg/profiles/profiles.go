//
// Copyright ⓒ 2023 Chakib Ben Ziane <contact@blob42.xyz> and [`GoSuki` contributors]
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

// Package profiles ...
package profiles

import (

	"github.com/blob42/gosuki/internal/logging"
	"github.com/blob42/gosuki/internal/utils"
)

var log = logging.GetLogger("profiles")

// ProfileManager is any module that can detect or list profiles, usually a browser module.
// One profile manager should be created for each browser flavour.
type ProfileManager interface {

	// Returns all profiles for a given flavour
	GetProfiles(flavour string) ([]*Profile, error)

	// If should watch all profiles
	WatchAllProfiles() bool

	// Notifies the module to use a custom profile
	UseProfile(p Profile) error

	// Returns all flavours supported by this module
	ListFlavours() []BrowserFlavour
}


type Profile struct {
	// Unique identifier for the profile
	Id   string

	// Name of the profile
	// This is usually the name of the directory where the profile is stored
	Name string

	// relative path to profile from base dir
	Path string

	// Base dir of the profile
	BaseDir string
}

func (f Profile) AbsolutePath() (string, error) {
	return utils.ExpandPath(f.BaseDir, f.Path)
}


// The BrowserFlavour struct stores the name of the browser and the base
// directory where the profiles are stored.
// Example flavours: chrome-stable, chrome-unstable, firefox, firefox-esr, librewolf, etc.
type BrowserFlavour struct {
	Name string
	BaseDir string
}

// Detect if the browser is installed. Returns true if the path exists
func (b BrowserFlavour) Detect() bool {
	var dir string
	var err error
	if dir, err = utils.ExpandPath(b.BaseDir); err != nil {
		log.Warningf("could not expand path <%s>: %s", b.BaseDir, err)
		return false
	} else if _, err = utils.CheckDirExists(dir); err != nil {
			log.Warningf("could not find browser <%s> at <%s>: %s", b.Name, dir, err)
			return false
		}

	return true
}

