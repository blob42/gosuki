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

package browsers

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func (f *BrowserFamily) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		switch strings.ToLower(value.Value) {
		case "mozilla":
			*f = Mozilla
		case "chrome":
			*f = ChromeBased
		case "qutebrowser":
			*f = Qutebrowser
		default:
			return fmt.Errorf("unknown family: %s", value.Value)
		}
	default:
		return fmt.Errorf("expected scalar node, got %v", value.Kind)
	}
	return nil
}

type flavour string

// top-level browser config struct for YAML parsing
type BrowserConfig struct {

	// Chrome based browsers
	Chrome map[flavour]Platforms `yaml:"chrome"`

	// Mozilla based browsers
	Mozilla map[flavour]Platforms `yaml:"mozilla"`

	// Custom BrowserDef
	Other map[BrowserFamily]CustomBrowser `yaml:"other"`
}

type platform string

type CustomBrowser map[flavour]Platforms

type Platforms map[platform]PlatformConfig

// Platform config data structure for generation process
type PlatformConfig struct {
	BaseDir string `yaml:"base_dir"`
	Snap    string `yaml:"snap"`
	Flatpak string `yaml:"flat"` // note: changed from flat to flatpak for clarity
}
