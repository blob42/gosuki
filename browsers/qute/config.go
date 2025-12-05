//
//  Copyright (c) 2024-2025 Chakib Ben Ziane <contact@blob42.xyz>  and [`gosuki` contributors](https://github.com/blob42/gosuki/graphs/contributors).
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

package qute

import (
	"github.com/blob42/gosuki/pkg/browsers"
	"github.com/blob42/gosuki/pkg/logging"
	"github.com/blob42/gosuki/pkg/modules"
)

var QuteBrowser = browsers.Defined(browsers.Qutebrowser)["qutebrowser"]

const (
	BrowserName    = "qutebrowser"
	DefaultProfile = "Default"
)

var (
	QuteCfg = NewQuteConfig()
	log     = logging.GetLogger("qute")
)

type QuteConfig struct {
	quickmarksPath         string `toml:"-"`
	*modules.BrowserConfig `toml:"-"`
	modules.ProfilePrefs   `toml:"profile_options" mapstructure:"profile_options"`
}

func NewQuteConfig() *QuteConfig {

	baseDir := QuteBrowser.GetBaseDir()

	config := &QuteConfig{
		quickmarksPath: baseDir + "/quickmarks",
		BrowserConfig: &modules.BrowserConfig{
			Name:           BrowserName,
			BkFile:         "urls",
			BkDir:          baseDir + "/bookmarks",
			BaseDir:        baseDir,
			UseFileWatcher: true,
			UseHooks:       []string{"bk_tags_from_name"},
		},
		ProfilePrefs: modules.ProfilePrefs{
			Profile: DefaultProfile,
		},
	}

	return config
}
