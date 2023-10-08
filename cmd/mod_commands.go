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

// commands related to modules
package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/blob42/gosuki/pkg/modules"
)

// map cmd Name to *cli.Command
type modCmds map[string]*cli.Command

var (
	// Map registered browser module IDs to their modCmds map
	modCommands = map[string]modCmds{}
)

// TODO: use same logic with browser mod registering
func RegisterModCommand(modID string, cmd *cli.Command) {
	if cmd == nil {
		log.Panicf("cannot register nil cmd for <%s>", modID)
	}

	if _, ok := modCommands[modID]; !ok {
		modCommands[modID] = make(modCmds)
	}
	modCommands[modID][cmd.Name] = cmd
}

// return list of registered commands for browser module
func RegisteredModCommands(modID string) modCmds {
	return modCommands[modID]
}

var ModuleCmds = &cli.Command {
	Name: "modules",
	Aliases: []string{"m"},
	Usage: "module commands",
	Subcommands: []*cli.Command{
		listModulesCmd,
	},
}

var listModulesCmd = &cli.Command{
	Name: "list",
	Aliases: []string{"l"},
	Usage: "list available browsers and modules",
	Action: func(_ *cli.Context) error {

		fmt.Printf("\n%s\n", "Modules:")
		mods := modules.GetModules()
		for _, mod := range mods {
			_, isBrowser := mod.(modules.BrowserModule)
			if isBrowser {
				fmt.Printf("-%-10s \t %s\n", mod.ModInfo().ID, "<browser>")
			} else {
				fmt.Printf("-%-10s \t %s\n", mod.ModInfo().ID, "<module>")
			}
		}
		return nil
	},
}
