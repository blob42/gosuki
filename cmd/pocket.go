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
	Name:      "pocket",
	Usage:     "import bookmarks from a Pocket export in csv format",
	Action:    importFromPocketCSV,
	ArgsUsage: "path/to/pocket-export.csv",
}

func importFromPocketCSV(ctx context.Context, c *cli.Command) error {
	if c.Args().Len() != 1 {
		return errors.New("missing path to csv file")
	}
	path := c.Args().Get(0)
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

		if len(row) < 6 {
			continue
		}

		url := row[1]
		title := row[2]
		// timeAdded := row[3]
		tags := row[4]
		desc := row[5]

		// Parse time_added as Unix timestamp (seconds)
		// sec, err := time.Parse("2006-01-02 15:04:05", timeAdded)
		// if err != nil {
		// 	sec, err = time.Parse("2006-01-02", timeAdded)
		// 	if err != nil {
		// 		continue
		// 	}
		// }

		bookmark := &gosuki.Bookmark{
			URL:    url,
			Title:  title,
			Tags:   strings.Split(tags, ","),
			Desc:   desc,
			Module: PocketImporterID,
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
