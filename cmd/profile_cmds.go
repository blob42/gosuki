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

package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/blob42/gosuki/internal/utils"
	"github.com/blob42/gosuki/pkg/modules"
	"github.com/blob42/gosuki/pkg/profiles"
)



var ProfileCmds = &cli.Command{
	Name: "profile",
	Aliases: []string{"p"},
	Usage: "profile commands",
	Subcommands: []*cli.Command{
		listProfilesCmd,
		detectInstalledCmd,
	},
}

//TODO: only enable commands when modules which implement profiles interfaces
// are available
var listProfilesCmd = &cli.Command{
	Name: "list",
	Usage: "list all available profiles",
	Action: func(c *cli.Context) error {

	browsers := modules.GetBrowserModules()
	for _, br := range browsers {

		//Create a browser instance
		brmod, ok := br.ModInfo().New().(modules.BrowserModule)
		if !ok {
			log.Criticalf("<%s> is not a BrowserModule", br.ModInfo().ID)
		}

		pm, isProfileManager := brmod.(profiles.ProfileManager)
		if !isProfileManager{
			log.Warningf("<%s> is not a profile manager", br.ModInfo().ID)
			continue
		}

		flavours := pm.ListFlavours()
		for _, f := range flavours {
			fmt.Printf("Profiles for <%s> flavour <%s>:\n\n", br.ModInfo().ID, f.Name)
			if profs, err := pm.GetProfiles(f.Name); err != nil {
				return err
			} else {
				for _, p := range profs {
					pPath, err := p.AbsolutePath()
					if err != nil {
						return err
					}
					fmt.Printf("%-10s[id:%s]\t %s\n", p.Name, p.Id, pPath)
				}
			}
			fmt.Println()
		}

	}

	return nil
	},
}


var detectInstalledCmd = &cli.Command{
	Name: "detect",
	Aliases: []string{"d"},
	Usage: "detect installed browsers",
	Action: func(_ *cli.Context) error {
		mods := modules.GetModules()
		fmt.Printf("installed browsers:\n\n")
		for _, mod := range mods {
			browser, isBrowser := mod.ModInfo().New().(modules.BrowserModule)
			if !isBrowser {
				log.Debugf("module <%s> is not a browser", mod.ModInfo().ID)
				continue
			}

			pm, isProf := browser.(profiles.ProfileManager)
			if !isProf {
				log.Debugf("module <%s> is not a profile manager", mod.ModInfo().ID)
				continue
			}

			flavours := pm.ListFlavours()
			for _, f := range flavours {
				log.Debugf("found flavour <%s> for <%s>", f.Name, mod.ModInfo().ID)
				if dir, err := utils.ExpandPath(f.BaseDir); err != nil {
					log.Errorf("could not expand path <%s> for flavour <%s>", f.BaseDir, f.Name)
					continue
				} else {
					f.BaseDir = dir
				}
				fmt.Printf("-%-10s \t %s\n", f.Name, f.BaseDir)
			}
		}

		return nil
	},
}
