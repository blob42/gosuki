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

// Package mozilla provides functionality for managing Mozilla-based browser profiles,
// such as Firefox and LibreWolf. It reads and parses the `profiles.ini` configuration file
// used by these applications to store profile information, and provides tools to retrieve
// and manage browser profiles for different flavors (e.g., Firefox, LibreWolf).
package mozilla

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/blob42/gosuki/internal/utils"
	"github.com/blob42/gosuki/pkg/browsers"
	"github.com/blob42/gosuki/pkg/logging"
	"github.com/blob42/gosuki/pkg/profiles"

	"github.com/go-ini/ini"
)

const (
	ProfilesFile = "profiles.ini"
)

type BrowserDef = browsers.BrowserDef

var (
	log           = logging.GetLogger("mozilla")
	ReIniProfiles = regexp.MustCompile(`(?i)profile`)

	ErrProfilesIni      = errors.New("could not parse profiles.ini file")
	ErrNoDefaultProfile = errors.New("no default profile found")
)

type MozProfileManager struct {
	PathResolver profiles.PathResolver
}

func NewMozProfileManager(resolver profiles.PathResolver) *MozProfileManager {

	return &MozProfileManager{
		PathResolver: resolver,
	}
}

func (pm *MozProfileManager) loadINIProfile(r profiles.PathResolver) (*ini.File, error) {
	log.Debugf("loading profile from <%s>", r.GetPath())
	profilePath, err := utils.ExpandPath(r.GetPath())
	if err != nil {
		return nil, err
	}

	pFile, err := ini.Load(profilePath)
	if err != nil {
		return nil, err
	}

	return pFile, nil
}

func (pm *MozProfileManager) GetProfiles(flavour string) ([]*profiles.Profile, error) {
	var pFile *ini.File
	var err error
	flv, ok := browsers.Defined(browsers.Mozilla)[flavour]

	if !ok {
		return nil, fmt.Errorf("unknown flavour <%s>", flavour)
	}

	baseDir, err := flv.ExpandBaseDir()
	if err != nil {
		return nil, fmt.Errorf("expanding base directory: %w", err)
	}

	pm.PathResolver.SetBaseDir(baseDir)
	if pFile, err = pm.loadINIProfile(pm.PathResolver); err != nil {
		return nil, err
	}

	sections := pFile.Sections()
	var result []*profiles.Profile
	for _, section := range sections {
		if ReIniProfiles.MatchString(section.Name()) {
			p := &profiles.Profile{
				ID:      section.Name(),
				BaseDir: baseDir,
			}

			err = section.MapTo(p)
			if err != nil {
				return nil, err
			}

			result = append(result, p)
		}
	}

	if len(result) == 0 {
		return nil, ErrProfilesIni
	}

	return result, nil
}

func (pm *MozProfileManager) GetProfileByName(flavour string, name string) (*profiles.Profile, error) {
	profs, err := pm.GetProfiles(flavour)
	if err != nil {
		return nil, err
	}

	for _, p := range profs {
		if p.Name == name {
			return p, nil
		}
	}

	return nil, fmt.Errorf("profile %s not found", name)
}

func (pm *MozProfileManager) ListFlavours() []BrowserDef {
	var result []BrowserDef

	// detect local flavours
	for _, v := range browsers.Defined(browsers.Mozilla) {
		if v.Detect() {
			result = append(result, *v)
		}
	}

	return result
}
