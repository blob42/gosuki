// Copyright (c) 2025 Chakib Ben Ziane <contact@blob42.xyz>  and [`gosuki` contributors](https://github.com/blob42/gosuki/graphs/contributors).
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
package database

import (
	"fmt"

	"github.com/blob42/gosuki"
)

type loadFunc func() ([]*gosuki.Bookmark, error)

// internal loading function called by modules
func LoadBookmarks(load loadFunc, modName string) error {
	var err error
	var buffer *DB

	// prepare buffer for module
	buffer, err = NewBuffer(modName)
	if err != nil {
		log.Errorf("could not create buffer for <%s>: %s", modName, err)
		return nil
	}
	defer buffer.Close()

	marks, err := load()
	if err != nil {
		log.Errorf("error fetching bookmarks: %s", err)
	}

	if len(marks) == 0 {
		log.Info("no bookmarks loaded", "module", modName)
		return nil
	}

	for _, mark := range marks {
		log.Trace("upserting", "bookmark", mark.URL)
		err = buffer.UpsertBookmark(mark)
		if err != nil {
			log.Errorf("db upsert: %s", mark.URL)
		}
	}
	//DEBUG:
	// buffer.DebugPrintBookmarks()
	err = buffer.SyncToCache()
	if err != nil {
		return fmt.Errorf("error syncing buffer to cache: %w", err)
	}

	ScheduleBackupToDisk()

	return nil
}
