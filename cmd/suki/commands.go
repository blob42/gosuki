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
	"errors"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/blob42/gosuki"
	"github.com/blob42/gosuki/internal/api"
	db "github.com/blob42/gosuki/internal/database"
)

type searchOpts struct {
	fuzzy bool
}

var FuzzySearchCmd = &cli.Command{
	Name:    "fuzzy",
	Aliases: []string{"f"},
	Usage:   "fuzzy search anywhere",
	UsageText: "Uses fuzzy search algorithm on any of the `URL`, `Title` and `Metadata`." +
		"Supports :tag syntax like search command for tag filtering.",
	Description: "",
	ArgsUsage:   "",
	Category:    "",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		if !cmd.Args().Present() {
			return errors.New("missing search term")
		}
		return searchBookmarks(ctx, cmd, searchOpts{true}, cmd.Args().Slice()...)
	},
}

var TagSearchCmd = &cli.Command{
	Name:    "search",
	Aliases: []string{"s"},
	Usage:   "search bookmarks by tags with AND/OR operators",
	UsageText: "suki search \"term :linux,kernel\" - searches for text + both tags\n" +
		"suki search \":OR linux kernel\" - searches for either tag (case-insensitive)",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return searchBookmarks(ctx, cmd, searchOpts{false}, cmd.Args().Slice()...)
	},
}

func formatMark(format string) (string, error) {
	outFormat := strings.Clone(format)

	// Comma separated list of tags
	outFormat = strings.ReplaceAll(outFormat, "%T", `{{ join .Tags "," }}`)

	// url
	outFormat = strings.ReplaceAll(outFormat, "%u", `{{.URL}}`)

	// title
	outFormat = strings.ReplaceAll(outFormat, "%t", `{{.Title}}`)

	// description
	outFormat = strings.ReplaceAll(outFormat, "%d", `{{.Desc}}`)

	r := strings.NewReplacer(`\t`, "\t", `\n`, "\n")
	outFormat = r.Replace(outFormat)

	return outFormat, nil
}

// Format a bookmark given a fmt.Printf format string
func formatPrint(_ context.Context, cmd *cli.Command, marks []*gosuki.Bookmark) error {
	for _, mark := range marks {
		if format := cmd.String("format"); format != "" {
			funcs := template.FuncMap{"join": strings.Join}
			outFormat, err := formatMark(format)
			if err != nil {
				return err
			}

			fmtTmpl, err := template.New("format").Funcs(funcs).Parse(outFormat)
			if err != nil {
				return err
			}

			err = fmtTmpl.Execute(os.Stdout, mark)
			if err != nil {
				return err
			}

		} else {
			fmt.Println(mark.URL)
		}
	}

	return nil
}

func listBookmarks(ctx context.Context, cmd *cli.Command) error {
	pageParms := db.PaginationParams{
		Page: 1,
		Size: -1,
	}
	result, err := db.ListBookmarks(ctx, &pageParms)
	if err != nil {
		return err
	}

	return formatPrint(ctx, cmd, result.Bookmarks)
}

func searchBookmarks(ctx context.Context, cmd *cli.Command, opts searchOpts, keyword ...string) error {
	if len(keyword) == 0 {
		return fmt.Errorf("no search keywords provided")
	}

	fullQuery := strings.Join(keyword, " ")
	query := api.ParseSearchQuery(fullQuery)

	var result *db.QueryResult
	var err error

	pageParms := db.PaginationParams{
		Page: 1,
		Size: -1,
	}

	if len(query.Tags) > 0 {
		result, err = db.QueryBookmarksByTags(
			ctx,
			query.TextQuery,
			query.Tags,
			query.TagCond,
			opts.fuzzy,
			&pageParms,
		)
	} else {
		result, err = db.QueryBookmarks(
			ctx,
			query.TextQuery,
			opts.fuzzy,
			&pageParms,
		)
	}

	if err != nil {
		return err
	}

	return formatPrint(ctx, cmd, result.Bookmarks)
}
