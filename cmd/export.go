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
	"html"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/blob42/gosuki"
	db "github.com/blob42/gosuki/internal/database"
)

var ExportCmds = &cli.Command{
	Name:        "export",
	Usage:       "One-time export to other formats",
	Description: `The export command provides functionality to export bookmarks to other browser or application formats. `,
	Commands: []*cli.Command{
		exportHTMLCmd,
	},
}

var exportHTMLCmd = &cli.Command{
	Name:        "html",
	Usage:       "Export bookmarks to Netscape bookmark format (HTML)",
	Description: `Exports all bookmarks to a file in Netscape bookmark format, which is compatible with most modern browsers.`,
	ArgsUsage:   "path/to/export.html",
	Action:      exportToHTML,
	Arguments: []cli.Argument{
		&cli.StringArg{
			Name:      "path",
			UsageText: "Export bookmarks to Netscape bookmark format (HTML), which is compatible with most modern browsers. The exported file can be imported into other applications that support this standard format.",
			Config: cli.StringConfig{
				TrimSpace: true,
			},
		},
	},
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "force",
			Aliases: []string{"f"},
			Usage:   "Overwrite existing files without prompting",
		},
	},
}

func exportToHTML(ctx context.Context, c *cli.Command) error {
	path := c.StringArg("path")

	if _, err := os.Stat(path); err == nil && !c.Bool("force") {
		return fmt.Errorf("file %s already exists. Use -f to overwrite", path)
	}

	db.Init(ctx, c)
	var rawResults db.RawBookmarks

	err := db.DiskDB.Handle.SelectContext(ctx,
		&rawResults,
		`SELECT * FROM gskbookmarks`)
	if err != nil {
		return err
	}

	htmlContent := generateNetscapeHTML(rawResults.AsBookmarks())

	if path == "-" {
		if _, err = fmt.Fprint(os.Stdout, htmlContent); err != nil {
			return err
		}
	} else {

		if err := os.WriteFile(path, []byte(htmlContent), 0644); err != nil {
			return fmt.Errorf("failed to write to %s: %w", path, err)
		}
	}

	return nil
}

func generateNetscapeHTML(bookmarks []*gosuki.Bookmark) string {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE NETSCAPE-Bookmark-file-1>
<META HTTP-EQUIV="Content-Type" CONTENT="text/html; charset=UTF-8">
<TITLE>Bookmarks</TITLE>
<H1>Bookmarks</H1>
<DL><p>
`)

	for _, b := range bookmarks {
		sb.WriteString(fmt.Sprintf(`    <DT><A HREF="%s" LAST_MODIFIED="%d">%s</A>
`,
			html.EscapeString(b.URL),
			b.Modified,
			html.EscapeString(b.Title),
		))
	}

	sb.WriteString("</DL>\n")
	return sb.String()
}
