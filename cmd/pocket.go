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
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/blob42/gosuki"
	db "github.com/blob42/gosuki/internal/database"
	"github.com/blob42/gosuki/internal/utils"
)

const (
	PocketImporterID = "pocket-import"
)

var importPocketCmd = &cli.Command{
	Name:  "pocket",
	Usage: "Import bookmarks from a Pocket export in CSV format",
	Description: `Import bookmarks from a Pocket CSV export file into the bookmarking system.

This command processes Pocket's exported CSV file and imports all bookmarks,
preserving their titles, URLs, and tags. The CSV file should be formatted
according to Pocket's standard export format.

The import will create a new bookmark entry for each URL in the CSV file,
maintaining the original metadata including tags and read/unread status.`,
	Action:    importFromPocketCSV,
	ArgsUsage: "path/to/pocket-export.csv",
	Arguments: []cli.Argument{
		&cli.StringArg{
			Name:      "path",
			UsageText: "Path to the Pocket CSV export file",
			Config: cli.StringConfig{
				TrimSpace: true,
			},
		},
	},
}

// CSV structure for Pocket import is:
//
// title,url,time_added,tags,status
// Opération Bobcat 1942-1946 - Tahiti Heritage,https://www.tahitiheritage.pf/operation-bobcat-1942-1946/,1689230329,history|military|polynesie|usa,unread
func importFromPocketCSV(ctx context.Context, c *cli.Command) error {
	path := c.StringArg("path")
	if c.StringArg("path") == "" {
		return errors.New("missing path to csv file")
	}
	expandedPath, err := utils.ExpandPath(path)
	if err != nil {
		return err
	}

	if _, err = os.Stat(expandedPath); os.IsNotExist(err) {
		return err
	}

	if !strings.HasSuffix(path, ".csv") {
		return errors.New("file does not end in .csv")
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fmt.Printf("importing from %s\n", path)

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV: %w", err)
	}

	db.Init(ctx, c)
	DB := db.DiskDB
	defer db.DiskDB.Close()

	var bkCount int
	for i, row := range records {
		// skip header
		if i == 0 {
			continue
		}

		if len(row) < 5 {
			continue
		}

		url := row[1]
		title := row[0]
		timeAdded := row[2]
		tags := row[4]

		modified, err := strconv.ParseUint(string(timeAdded), 10, 64)
		if err != nil {
			panic(err)
		}

		bookmark := &gosuki.Bookmark{
			URL:      url,
			Title:    title,
			Tags:     strings.Split(tags, "|"),
			Module:   PocketImporterID,
			Modified: modified,
		}

		if err = DB.UpsertBookmark(bookmark); err != nil {
			fmt.Fprintf(os.Stderr, "inserting bookmark %s: %s", bookmark.URL, err)
			continue
		} else {
			bkCount++
		}
	}
	fmt.Printf("imported %d bookmarks\n", bkCount)

	return nil
}
