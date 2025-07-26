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

// Main command line entry point for gosuki
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/blob42/gosuki/internal/utils"
	"github.com/blob42/gosuki/pkg/build"
	"github.com/blob42/gosuki/pkg/config"
	"github.com/blob42/gosuki/pkg/logging"
	"github.com/blob42/gosuki/pkg/modules"

	"github.com/urfave/cli/v3"

	"github.com/blob42/gosuki/cmd"

	_ "github.com/blob42/gosuki/browsers/chrome"
	_ "github.com/blob42/gosuki/browsers/firefox"
	_ "github.com/blob42/gosuki/browsers/qute"

	_ "github.com/blob42/gosuki/mods"
)

var (
	log = logging.GetLogger("MAIN")
)

func main() {

	app := cli.Command{}

	app.Name = "gosuki"
	app.Description = "TODO: summary gosuki description"
	app.Usage = "GoSuki watches and saves your bookmraks across all installed browsers in realtime."
	app.Version = build.Version()
	app.Suggest = true
	app.EnableShellCompletion = true
	app.ExitErrHandler = func(ctx context.Context, cli *cli.Command, err error) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		} else {
			os.Exit(0)
		}
	}

	flags := []cli.Flag{

		&cli.StringFlag{
			Name:        "config",
			Aliases:     []string{"c"},
			Value:       config.DefaultConfPath(),
			Usage:       "config `path`",
			DefaultText: "~/.config/gosuki/config.toml",
			Category:    "_",
		},

		&cli.BoolFlag{
			Name:     "tui",
			Aliases:  []string{"T"},
			Category: "_",
			Value:    false,
		},

		&cli.IntFlag{
			Name:        "debug",
			Aliases:     []string{"D"},
			Category:    "_",
			DefaultText: "0",
			Usage:       "set debug `level` (-1..3)",
			Sources:     cli.EnvVars(logging.EnvGosukiDebug),
			Action: func(_ context.Context, _ *cli.Command, val int) error {
				if logging.SilentMode {
					logging.SetLogLevel(-1)
				} else {
					logging.SetLogLevel(val)
				}
				return nil
			},
		},

		&cli.BoolFlag{
			Name:        "silent",
			Aliases:     []string{"S"},
			Category:    "_",
			DefaultText: "0",
			Usage:       "disable all log output",
			Action: func(_ context.Context, _ *cli.Command, val bool) error {
				if val {
					logging.SetLogLevel(-1)
				}
				return nil
			},
		},
	}

	flags = append(flags, config.SetupGlobalFlags()...)
	app.Flags = append(app.Flags, flags...)

	app.Before = func(ctx context.Context, c *cli.Command) (context.Context, error) {

		// The order here is important
		//
		// 1. load the file config
		// NOTE: since the parsing of flags is done later, the default config
		// will not take into consideration the cli flags when generating the
		// file.
		// 2. every module has the opprtunity to register its own flags
		// 3. the modules can run custom code before the cli is ready but after
		// the config is ready, using the config hooks.
		//
		// Cli flags have the highest priority and override config file values

		config.Init(c.String("config"))
		if logging.SilentMode {
			logging.SetLogLevel(-1)
		} else {
			logging.SetLogLevel(logging.DefaultLogLevels[logging.LoggingMode])
		}

		// get all registered browser modules
		modules := modules.GetModules()
		for _, mod := range modules {

			// Run module's hooks that should run before context is ready
			// for example setup flags management
			modinfo := mod.ModInfo()
			hook := cmd.BeforeHook(string(modinfo.ID))
			if hook != nil {
				if err := cmd.BeforeHook(string(modinfo.ID))(ctx, c); err != nil {
					return ctx, err
				}
			}
		}

		// Execute config hooks
		// DOC: better documentation of Conf hooks
		// modules can run custom code before the CLI is ready.
		// For example read the environment and set configuration options to be
		// used by the module instances.
		config.RunConfHooks(ctx, c)

		return ctx, nil
	}

	// Browser modules can register commands through cmd.RegisterModCommand.
	// registered commands will be appended here
	app.Commands = []*cli.Command{
		// main entry point
		startDaemonCmd,
		cmd.ConfigCmds,
		cmd.DetectCmd,
		cmd.ProfileCmds,
		cmd.ModuleCmds,
		cmd.DebugCmd,
		cmd.ImportCmds,
	}

	// Add global flags from registered modules
	// we use GetModules to handle all types of modules
	modules := modules.GetModules()

	log.Debugf("loading %d modules", len(modules))
	for _, mod := range modules {
		modID := string(mod.ModInfo().ID)
		log.Debugf("loading module <%s>", modID)

		// for each registered module, register own flag management
		modFlags := cmd.GlobalFlags(modID)
		if len(modFlags) != 0 {
			app.Flags = append(app.Flags, modFlags...)
		}

		// Add all browser module registered commands
		cmds := cmd.RegisteredModCommands(modID)
		for _, cmd := range cmds {
			app.Commands = append(app.Commands, cmd)
		}
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func init() {
	if err := utils.MkGosukiDataDir(); err != nil {
		log.Fatal(err)
	}
}
