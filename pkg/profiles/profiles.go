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

// Package profiles ...
package profiles

import (
	"fmt"
	"path/filepath"

	"github.com/blob42/gosuki/internal/utils"
	"github.com/blob42/gosuki/pkg/browsers"
	"github.com/blob42/gosuki/pkg/logging"
)

var log = logging.GetLogger("profiles")

type BrowserDef = browsers.BrowserDef

type Profile struct {
	// Unique identifier for the profile
	ID string

	// Name of the profile
	// This is usually the name of the directory where the profile is stored
	Name string

	// path to profile
	// relative when IsRelative is true
	Path string

	IsRelative bool

	// Base dir of the profile
	BaseDir string

	IsCustom bool
}

// returns shortcut for path
func (p Profile) ShortBaseDir() string {
	if !p.IsRelative {
		return ""
	}
	return utils.Shorten(p.BaseDir)
}

// ProfileManager is any module that can detect or list profiles, usually a browser module.
// One profile manager should be created for each browser flavour.
type ProfileManager interface {

	// Returns all profiles for a given flavour
	GetProfiles(flavour string) ([]*Profile, error)

	// If should watch all profiles
	WatchAllProfiles() bool

	// Notifies the module to use a custom profile and flavour
	UseProfile(p *Profile, f *BrowserDef) error

	// Get current active profile
	GetProfile() *Profile

	// Returns all flavours supported by this module
	ListFlavours() []BrowserDef

	// Get current active flavour
	GetCurFlavour() *BrowserDef
}

func FromCustom(list []CustomProfile, flavour string) []*Profile {
	var result []*Profile

	for _, cp := range list {
		if cp.Flavour == flavour {
			result = append(result, &Profile{
				Name:     cp.Name,
				ID:       cp.Name,
				Path:     cp.Path,
				BaseDir:  filepath.Join(cp.Path, "../"),
				IsCustom: true,
			})
		}
	}

	return result
}

type CustomProfile struct {
	Name    string
	Path    string
	Flavour string
}

// GetFlavour Returns flavour of browser given a normalized browser base dir
// TEST:
// DEAD:
func GetFlavour(pm ProfileManager, baseDir string) string {
	flavours := pm.ListFlavours()
	for _, f := range flavours {
		bDir, err := f.ExpandBaseDir()
		if err != nil {
			log.Error("expanding basedir", "flavour", f.Flavour, "err", err)
			return ""
		}

		if bDir == baseDir {
			return f.Flavour
		}
	}
	return ""
}

func (p Profile) AbsolutePath() (string, error) {
	log.Debugf("Profile debug: Name=%s, IsRelative=%t, Path=%s, BaseDir=%s",
		p.Name, p.IsRelative, p.Path, p.BaseDir)

	if p.IsRelative {
		if p.BaseDir == "" {
			return "", fmt.Errorf("profile baseDir is empty for relative profile %s", p.Name)
		}
		if p.Path == "" {
			return "", fmt.Errorf("profile path is empty for relative profile %s", p.Name)
		}
		return utils.ExpandPath(p.BaseDir, p.Path)
	}

	if p.Path == "" {
		return "", fmt.Errorf("profile path is empty for absolute profile %s", p.Name)
	}
	return utils.ExpandPath(p.Path)
}
