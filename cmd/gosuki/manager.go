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

//go:build (systray && darwin) || (systray && windows) || !systray

package main

import (
	"fmt"
	"os"

	"github.com/blob42/gosuki/internal/server"
	"github.com/blob42/gosuki/internal/webui"
	"github.com/blob42/gosuki/pkg/manager"
	"github.com/blob42/gosuki/pkg/modules"
)

func initManager(tuiMode bool) *manager.Manager {
	manager := manager.NewManager()
	manager.ShutdownOn(os.Interrupt)

	uiServ := server.NewWebUIServer(tuiMode)
	manager.AddUnit(uiServ, fmt.Sprintf("webui[%s]", webui.BindAddr))

	manager.AddUnit(&modules.MsgDispatcher, modules.DispatcherID).SetRecoverable()

	return manager
}
