// Copyright (c) 2023 Chakib Ben Ziane <contact@blob42.xyz>  and [`gosuki` contributors](https://github.com/blob42/gosuki/graphs/contributors).
// All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This file is part of GoSuki.
//
// GoSuki is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// GoSuki is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with gosuki.  If not, see <http://www.gnu.org/licenses/>.
package config

import "github.com/urfave/cli/v2"

// Setup cli flag for global options
func SetupGlobalFlags() []cli.Flag {
	log.Debugf("Setting up global flags")
	flags := []cli.Flag{}
	for k, v := range configs[GlobalConfigName].Dump() {
		log.Debugf("Registering global flag %s = %v", k, v)

		// Register the option as a cli flag
		switch val := v.(type) {
			case string:
				flags = append(flags, &cli.StringFlag{
					Category: "_",
					Name:  k,
					Value: val,
				})

			case int:
				flags = append(flags, &cli.IntFlag{
					Category: "_",
					Name: k,
					Value: val,
				})

			case bool:
				flags = append(flags, &cli.BoolFlag{
					Category: "_",
					Name: k,
					Value: val,
				})

			default:
				log.Fatalf("unsupported type for global option %s", k)
		}
	}

	return flags
}


//TODO: setup module flags
