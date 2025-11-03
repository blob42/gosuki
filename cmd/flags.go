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

package cmd

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	altsrc "github.com/urfave/cli-altsrc/v3"
	"github.com/urfave/cli-altsrc/v3/toml"
	"github.com/urfave/cli/v3"

	"github.com/blob42/gosuki/internal/database"
	"github.com/blob42/gosuki/internal/utils"
	"github.com/blob42/gosuki/internal/webui"
	"github.com/blob42/gosuki/pkg/config"
	"github.com/blob42/gosuki/pkg/logging"
)

var MainFlags = []cli.Flag{
	logging.DebugFlag,
	&cli.StringFlag{
		Name:        "config",
		Aliases:     []string{"c"},
		Value:       config.DefaultConfPath(),
		Usage:       "config `path`",
		DefaultText: utils.Shorten(config.DefaultConfPath()),
		Destination: &config.ConfigFileFlag,
	},

	&cli.StringFlag{
		Name:        "db",
		Value:       database.GetDBPath(),
		DefaultText: utils.Shorten(database.GetDBPath()),
		Usage:       "`path` where gosuki.db is stored",
		Destination: &config.DBPath,
		Sources:     cli.NewValueSourceChain(toml.TOML("database.path", altsrc.NewStringPtrSourcer(&config.ConfigFileFlag))),
	},

	&cli.StringFlag{
		Name:        "listen",
		Aliases:     []string{"l"},
		Value:       webui.DefaultBindAddr(),
		Usage:       "listening address (host:port)",
		Destination: &webui.BindAddr,
		Validator: func(s string) error {
			var ip net.IP
			var nPort int
			var err error
			parts := strings.Split(s, ":")
			if len(parts) != 2 {
				return fmt.Errorf("invalid listen format, expected host:port")
			}
			host := parts[0]
			port := parts[1]

			if ip = net.ParseIP(host); ip == nil {
				// If not an IP address, treat as hostname
				return fmt.Errorf("invalid ip: %s", host)
			}

			if nPort, err = strconv.Atoi(port); err != nil {
				return fmt.Errorf("invalid port: %w", err)
			}

			webui.BindHost = ip.To4().String()
			webui.BindPort = nPort
			return nil
		},

		// it should be split and parsed instead
		Sources: cli.NewValueSourceChain(toml.TOML("webui.listen", altsrc.NewStringPtrSourcer(&config.ConfigFileFlag))),
	},
}
