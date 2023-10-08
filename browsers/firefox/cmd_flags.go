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

package firefox

import (
	"strings"

	"github.com/blob42/gosuki/cmd"
	"github.com/blob42/gosuki/internal/config"
	"github.com/blob42/gosuki/internal/utils"

	"github.com/gobuffalo/flect"
	"github.com/urfave/cli/v2"
)

const (
	FirefoxProfileFlag = "ff-profile"
)

var globalFirefoxFlags = []cli.Flag{
    // This allows us to register dynamic cli flags which get converted to 
    // config.Configurator options. 
    // The flag must be given a name in the form `--firefox-<flag>`.
	&cli.StringFlag{
		Name:  FirefoxProfileFlag,
		Category: "firefox",
		Usage: "set the default firefox `PROFILE` to use",
	},
	&cli.BoolFlag{
		Name:        "ff-watch-all-profiles",
		Category:	 "firefox",
		Usage:       "watch all firefox profiles for changes", 
		Aliases:     []string{"ff-watch-all"},
	},
}

// Firefox global flags must start with --firefox-<flag name here>
// NOTE: is called in *cli.App.Before callback
func globalCommandFlagsManager(c *cli.Context) error {
	log.Debugf("<%s> registering global flag manager", BrowserName)
	for _, f := range c.App.Flags {

		if utils.InList(f.Names(), "help") ||
			utils.InList(f.Names(), "version") {
			continue
		}

		if !c.IsSet(f.Names()[0]) {
			continue
		}

		sp := strings.Split(f.Names()[0], "-")

		if len(sp) < 2 {
			continue
		}

		// TEST:
		// TODO: add doc
		// Firefox flags must start with --firefox-<flag name here>
		// or -ff-<flag name here>
		if !utils.InList([]string{"firefox", "ff"}, sp[0]) {
			continue
		}

		//TODO: document this feature
        // extracts global options that start with --firefox-*
		optionName := flect.Pascalize(strings.Join(sp[1:], " "))
		var destVal interface{}

		// Find the corresponding flag
		for _, ff := range globalFirefoxFlags {
			if ff.String() == f.String() {

				// Type switch on the flag type
				switch ff.(type) {

				case *cli.StringFlag:
					destVal = c.String(f.Names()[0])

				case *cli.BoolFlag:
					destVal = c.Bool(f.Names()[0])
				}

			}
		}

		err := config.RegisterModuleOpt(BrowserName,
			optionName, destVal)

		if err != nil {
			log.Panic(err)
		}
	}
	return nil
}

func init() {
	// register dynamic flag manager for firefox
	cmd.RegBeforeHook(BrowserName, globalCommandFlagsManager)

    for _, flag := range globalFirefoxFlags {
        cmd.RegGlobalModFlag(BrowserName, flag)
    }
}
