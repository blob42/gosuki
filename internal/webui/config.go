// Copyright (c) 2025 Chakib Ben Ziane <contact@blob42.xyz>  and [`gosuki` contributors](https://github.com/blob42/gosuki/graphs/contributors).
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
package webui

import (
	"fmt"

	"github.com/blob42/gosuki/pkg/config"
)

var (
	BindHost = "127.0.0.1"
	BindPort = 2025
	Config   *webuiConf
	BindAddr string
)

type webuiConf struct {
	Listen string `toml:"listen" mapstructure:"listen"`
}

func DefaultBindAddr() string {
	return fmt.Sprintf("%s:%d", BindHost, BindPort)
}

func init() {
	Config = &webuiConf{
		Listen: DefaultBindAddr(),
	}

	config.RegisterConfigurator("webui", config.AsConfigurator(Config))
}
