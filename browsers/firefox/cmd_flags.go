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

package firefox

import (
	"context"
	"strings"

	"github.com/blob42/gosuki/cmd"
	"github.com/blob42/gosuki/internal/utils"
	"github.com/blob42/gosuki/pkg/config"

	"github.com/gobuffalo/flect"
	"github.com/urfave/cli/v3"
)

const (
	FirefoxProfileFlag = "ff-profile"
)

var globalFirefoxFlags = []cli.Flag{
	// This allows us to register dynamic cli flags which get converted to
	// config.Configurator options.
	// The flag must be given a name in the form `--firefox-<flag>` or `--ff-<flag>`.
	&cli.StringFlag{
		Name:     FirefoxProfileFlag,
		Category: "firefox",
		Usage:    "set the default firefox `PROFILE` to use",
	},
}

// Firefox global flags must start with --firefox-<flag name here>
// NOTE: is called in *cli.App.Before callback
// TEST: cli flag parsing
func globalCommandFlagsManager(ctx context.Context, cmd *cli.Command) error {
	log.Debugf("<%s> registering global flag manager", BrowserName)
	for _, f := range cmd.Flags {

		if utils.InList(f.Names(), "help") ||
			utils.InList(f.Names(), "version") {
			continue
		}

		if !cmd.IsSet(f.Names()[0]) {
			continue
		}

		sp := strings.Split(f.Names()[0], "-")

		if len(sp) < 2 {
			continue
		}

		// TEST:
		// Firefox flags must start with --firefox-<flag name here>
		// or -ff-<flag name here>
		if !utils.InList([]string{"firefox", "ff"}, sp[0]) {
			continue
		}

		// extracts global options that start with --firefox-*
		optionName := flect.Pascalize(strings.Join(sp[1:], " "))
		var destVal interface{}

		// Find the corresponding flag
		for _, ff := range globalFirefoxFlags {
			if ff.String() == f.String() {

				// Type switch on the flag type
				switch ff.(type) {

				case *cli.StringFlag:
					destVal = cmd.String(f.Names()[0])

				case *cli.BoolFlag:
					destVal = cmd.Bool(f.Names()[0])
				}

			}
		}

		err := config.RegisterModuleOpt(BrowserName,
			optionName, destVal)

		if err != nil {
			log.Fatal(err)
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
