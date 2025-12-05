//
//  Copyright (c) 2025 Chakib Ben Ziane <contact@blob42.xyz>  and [`gosuki` contributors](https://github.com/blob42/gosuki/graphs/contributors).
//  All rights reserved.
//
//  SPDX-License-Identifier: AGPL-3.0-or-later
//
//  This file is part of GoSuki.
//
//  GoSuki is free software: you can redistribute it and/or modify
//  it under the terms of the GNU Affero General Public License as
//  published by the Free Software Foundation, either version 3 of the
//  License, or (at your option) any later version.
//
//  GoSuki is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of
//  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU Affero General Public License for more details.
//
//  You should have received a copy of the GNU Affero General Public License
//  along with gosuki.  If not, see <http://www.gnu.org/licenses/>.
//

//go:generate go run ./gen browsers.yaml
package browsers

import "github.com/blob42/gosuki/internal/utils"

type BrowserFamily uint

const (
	Mozilla BrowserFamily = iota
	ChromeBased
	Qutebrowser
)

type BrowserDef struct {
	Flavour string // also acts as canonical name

	Family BrowserFamily // browser family

	// Base browser directory path
	BaseDir string

	// (linux only) path to snap package base dir
	SnapDir string

	// (linux only) path to flatpak package base dir
	FlatpakDir string
}

func (b BrowserDef) Detect() bool {
	var dir string
	var err error
	if dir, err = b.ExpandBaseDir(); err != nil {
		log.Debugf("expand path: %s: %s", b.GetBaseDir(), err)
		log.Info("skipping", "flavour", b.Flavour)
	} else if ok, err := utils.DirExists(dir); err != nil || !ok {
		log.Infof("could not detect <%s>: %s: %s", b.Flavour, dir, err)
		return false
	}

	return true
}

func MozBrowser(flavour, base, snap, flat string) BrowserDef {
	return BrowserDef{
		Flavour:    flavour,
		BaseDir:    base,
		Family:     Mozilla,
		SnapDir:    snap,
		FlatpakDir: flat,
	}
}

func ChromeBrowser(flavour, base, snap, flat string) BrowserDef {
	return BrowserDef{
		Flavour:    flavour,
		BaseDir:    base,
		Family:     ChromeBased,
		SnapDir:    snap,
		FlatpakDir: flat,
	}
}
