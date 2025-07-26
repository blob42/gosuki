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

//go:build darwin && systray

package main

import (
	"context"
	"os"

	"github.com/blob42/gosuki/internal/gui"
	"github.com/blob42/gosuki/internal/utils"
	"github.com/blob42/gosuki/pkg/logging"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v3"
)

func startDaemon(ctx context.Context, cmd *cli.Command) error {
	defer utils.CleanFiles()

	// initialize webui and non module units

	tuiOpts := []tea.ProgramOption{
		// tea.WithAltScreen(),
	}

	//TUI MODE
	if cmd.Bool("tui") && isatty.IsTerminal(os.Stdout.Fd()) {
		manager := initManager(true)

		tui := NewTUI(func(tea.Model) tea.Cmd {
			return func() tea.Msg {
				err := startNormalDaemon(ctx, cmd, manager)
				if err != nil {
					return ErrMsg(err)
				}
				return DaemonStartedMsg{}
			}
		}, manager, tuiOpts...)

		logging.SetTUI(tui.model.logBuffer)
		return tui.Run()
	}

	manager := initManager(false)

	startNormalDaemon(ctx, cmd, manager)
	gui.RunSystray(manager)
	<-manager.Quit
	return nil
}
