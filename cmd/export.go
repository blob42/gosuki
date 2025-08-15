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
	"context"
	"fmt"
	"io"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/urfave/cli/v3"

	db "github.com/blob42/gosuki/internal/database"
	"github.com/blob42/gosuki/pkg/export"
)

var ExportCmds = &cli.Command{
	Name:        "export",
	Usage:       "One-time export to other formats",
	Description: `The export command provides functionality to export bookmarks to other browser or application formats. `,
	Commands: []*cli.Command{
		exportNSHTMLCmd,
		exportPocketHTMLCmd,
		exportJSONCmd,
		exportRSSCmd,
	},
}

var overwriteFlag = &cli.BoolFlag{
	Name:    "force",
	Aliases: []string{"f"},
	Usage:   "Overwrite existing files without prompting",
}

var exportNSHTMLCmd = &cli.Command{
	Name:        "html",
	Usage:       "Export bookmarks to Netscape bookmark format (HTML)",
	Description: `Exports all bookmarks to a file in Netscape bookmark format, which is compatible with most modern browsers.`,
	ArgsUsage:   "path/to/export.html",
	Action:      exportToFormat(export.NetscapeHTML),
	Arguments: []cli.Argument{
		&cli.StringArg{
			Name:      "path",
			UsageText: "Export bookmarks to Netscape bookmark format (HTML), which is compatible with most modern browsers. The exported file can be imported into other applications that support this standard format.",
			Config: cli.StringConfig{
				TrimSpace: true,
			},
		},
	},
	Flags: []cli.Flag{overwriteFlag},
}

var exportJSONCmd = &cli.Command{
	Name:        "json",
	Usage:       "Export bookmarks to JSON format (Pinboard/Wallabag)",
	Description: `Exports all bookmarks to a file in JSON format compatible with Pinboard and Wallabag.`,
	ArgsUsage:   "path/to/export.json",
	Action:      exportToFormat(export.JSON),
	Arguments: []cli.Argument{
		&cli.StringArg{
			Name:      "path",
			UsageText: "Export bookmarks to JSON format (Pinboard/Wallabag). The exported file can be imported into Pinboard, Wallabag and other applications that support this standard format.",
			Config: cli.StringConfig{
				TrimSpace: true,
			},
		},
	},
	Flags: []cli.Flag{overwriteFlag},
}

var exportRSSCmd = &cli.Command{
	Name:        "rss",
	Usage:       "Export bookmarks to generic RSS XML format",
	Description: `Exports all bookmarks to a generic RSS XML file, which can be imported into applications that support this standard format.`,
	ArgsUsage:   "path/to/export.rss",
	Action:      exportToFormat(export.RSS),
	Arguments: []cli.Argument{
		&cli.StringArg{
			Name:      "path",
			UsageText: "Export bookmarks to RSS XML format. The exported file can be imported into applications that support this standard format.",
			Config: cli.StringConfig{
				TrimSpace: true,
			},
		},
	},
	Flags: []cli.Flag{overwriteFlag},
}

// exports to pocket export html file format
var exportPocketHTMLCmd = &cli.Command{
	Name:        "pocket-html",
	Usage:       "Export bookmarks to Pocket HTML format",
	Description: `Exports all bookmarks to a file in Pocket HTML format.`,
	ArgsUsage:   "path/to/export.html",
	Action:      exportToFormat(export.PocketHTML),
	Arguments: []cli.Argument{
		&cli.StringArg{
			Name:      "path",
			UsageText: "Export bookmarks to Pocket HTML format. The exported file can be imported into Pocket and other applications that support this standard format.",
			Config: cli.StringConfig{
				TrimSpace: true,
			},
		},
	},
	Flags: []cli.Flag{overwriteFlag},
}

func exportToFormat(format int) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		var rows *sqlx.Rows
		var exporter export.Exporter
		var output io.WriteCloser
		var err error

		path := cmd.StringArg("path")
		if path == "" {
			return fmt.Errorf("missing path: ... export %s", cmd.ArgsUsage)
		}

		if _, err = os.Stat(path); err == nil && !cmd.Bool("force") {
			return fmt.Errorf("file %s already exists. Use -f to overwrite", path)
		}

		db.Init(ctx, cmd)
		if rows, err = db.DiskDB.Handle.QueryxContext(
			ctx,
			`SELECT * FROM gskbookmarks`,
		); err != nil {
			return err
		}

		switch format {
		case export.NetscapeHTML:
			exporter = &export.NetscapeHTMLExporter{}
		case export.PocketHTML:
			exporter = &export.PocketHTMLExporter{}
		case export.JSON:
			exporter = &export.JSONExporter{}
		case export.RSS:
			exporter = &export.RSSXMLExporter{}
		default:
			panic(fmt.Sprintf("unsupported export format %#v", format))
		}

		if path == "-" {
			output = os.Stdout
		} else {
			output, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				return err
			}
			defer output.Close()
		}
		bookExporter := export.NewBookmarksExporter(exporter, output)
		if format == export.JSON {
			bookExporter.Separator = ","
		}
		return bookExporter.ExportFromRows(rows)
	}

}
