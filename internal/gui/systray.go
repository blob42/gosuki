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

//go:build systray

package gui

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/energye/systray"
	"github.com/godbus/dbus/v5"
	"github.com/skratchdot/open-golang/open"

	"github.com/blob42/gosuki/internal/gui/icon"
	"github.com/blob42/gosuki/internal/utils"
	"github.com/blob42/gosuki/internal/webui"
	"github.com/blob42/gosuki/pkg/manager"
)

type Systray struct{}

func onReady() {
	systray.SetTemplateIcon(icon.Data, icon.Data)
	if runtime.GOOS != "darwin" {
		systray.SetTitle("GoSuki")
	}
	systray.SetTooltip("GoSuki Bookmark Manager")

	mUI := systray.AddMenuItem("Web UI", "Local Web UI")
	mUI.Click(func() {
		println(fmt.Sprintf("http://127.0.0.1:%d", webui.BindPort))
		open.Run(fmt.Sprintf("http://127.0.0.1:%d", webui.BindPort))
	})

	systray.AddSeparator()

	mHelp := systray.AddMenuItem("Help", "Open Help Page")
	mHelp.Click(func() {
		open.Run("http://gosuki.net/docs")
	})

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")
	mQuit.Click(func() {
		systray.Quit()
	})

	systray.SetOnClick(func(menu systray.IMenu) {
		open.Run(fmt.Sprintf("http://127.0.0.1:%d", webui.BindPort))
	})

	//NOTE: this is not required, it just allows to mutate the systray after
	//starting main callbacks
	// We can manipulate the systray in other goroutines
	// go func() {
	// 	mUrl := systray.AddMenuItem("Open UI", "GoSuki Web UI")
	// 	mUrl.Click(func() {
	// 		open.Run("https://blob42.xyz")
	// 	})
	// }()
}

func DarwinRunSystray(m *manager.Manager) {
	// ugly way to handle exit signal without blocking the gui thread
	go func() {
		<-m.Quit
		utils.CleanFiles()
		os.Exit(0)
	}()
	systray.Run(onReady, func() { m.Shutdown() })
}

func (st *Systray) Run(m manager.UnitManager) {
	// Guard: energye/systray silently ignores D-Bus connection errors and
	// panics on nil conn when ExportStatusNotifierItem is called.
	// In headless environments (containers, no display) there is no session bus.
	// This check runs synchronously in the unit's goroutine (before spawning
	// child goroutines) so the manager's recover in runUnit can catch any
	// unexpected panic, and the unit exits cleanly via Done().
	if runtime.GOOS == "linux" {
		conn, err := dbus.ConnectSessionBus()
		if err != nil {
			log.Printf("gui: skipping systray: no D-Bus session bus available (%v)", err)
			<-m.ShouldStop()
			m.Done()
			return
		}
		conn.Close()
	}

	onExit := func() {
		m.RequestShutdown()
	}

	go func() {
		systray.Run(onReady, onExit)
	}()

	<-m.ShouldStop()
	m.Done()
}
