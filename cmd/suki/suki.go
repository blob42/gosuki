// Copyright (c) 2024-2025-2025-2025-2025 Chakib Ben Ziane <contact@blob42.xyz>  and [`gosuki` contributors](https://github.com/blob42/gosuki/graphs/contributors).
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

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/blob42/gosuki/cmd"
	db "github.com/blob42/gosuki/internal/database"
	"github.com/blob42/gosuki/pkg/build"
	"github.com/blob42/gosuki/pkg/config"
	"github.com/blob42/gosuki/pkg/logging"
)

func main() {
	app := cli.Command{}
	app.Version = build.Version()

	app.Name = "suki"
	app.Usage = "gosuki lightweight cli - the universal bookmark manager"
	app.Description = `
suki is a lightweight, efficient command-line interface designed for querying and managing
bookmarks stored in gosuki. It offers fast, streamlined access to your bookmark collection
while maintaining minimal resource usage, making it ideal for frequent use in terminal environments.

The default output format is fully compatible with dmenu and other dmenu-compatible programs,
enabling seamless integration into your workflow through piping. Additionally, suki supports
customizable output formatting through the -F flag, allowing you to tailor the display to your specific needs.

Usage examples:
  suki                    # Display all bookmarks in dmenu-compatible format
  suki -f "%u | %t"       # Show only bookmark urls 
  suki "search term"      # Search for specific bookmarks
  suki | dmenu            # Pipe output to dmenu for interactive selection`
	app.UsageText = "suki [OPTIONS] [KEYWORD [KEYWORD...]] "
	app.HideVersion = true
	app.EnableShellCompletion = true
	app.Suggest = true
	app.CustomRootCommandHelpTemplate = AppHelpTemplate

	app.Flags = []cli.Flag{

		&cli.StringFlag{
			Name:    "format",
			Usage:   "Format output using a custom template",
			Aliases: []string{"f"},
		},
	}
	app.Flags = append(app.Flags, cmd.MainFlags...)

	app.Before = func(ctx context.Context, c *cli.Command) (context.Context, error) {
		config.Init(c.String("config"))
		db.RegisterSqliteHooks()
		err := db.InitDiskConn(config.DBPath)
		if _, isDBErr := err.(db.DBError); isDBErr {
			fmt.Fprintln(os.Stderr, "Database initialization failed:", err)
			fmt.Fprintln(os.Stderr, "Please ensure you have run `gosuki start` to initialize the database")
			os.Exit(10)
		}

		return ctx, err
	}

	app.Commands = []*cli.Command{
		FuzzySearchCmd,
		TagSearchCmd,
	}

	app.ExitErrHandler = func(ctx context.Context, cli *cli.Command, err error) {
		if err != nil {
			if err == logging.ErrHelpQuit {
				os.Exit(0)
			}
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		} else {
			os.Exit(0)
		}
	}

	app.Action = func(ctx context.Context, cmd *cli.Command) error {
		// if no argument passed, list all bookmarks
		if cmd.Args().Len() == 0 {
			return listBookmarks(ctx, cmd)
		}

		// use ~ as fuzzy character
		firstKw := cmd.Args().First()
		opts := searchOpts{}

		if firstKw[0] == '~' {
			opts.fuzzy = true
			firstKw = firstKw[1:]
		}

		return searchBookmarks(ctx, cmd, opts, firstKw)
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
