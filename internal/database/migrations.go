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

package database

// v2 adds the xhsum column to `gskbookmarks` table and calculates the xhsum
// for all bookmarks.
func (db *DB) migrateToVersion2() error {
	log.Debug("DB schema: migrating to v2")
	tx, err := db.Handle.Begin()
	if err != nil {
		return err
	}

	// add column xhsum if missing
	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM pragma_table_info('gskbookmarks') WHERE name = 'xhsum'").Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		_, err = tx.Exec("ALTER TABLE gskbookmarks ADD COLUMN xhsum TEXT DEFAULT ''")
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// Calculate xhsum for existing bookmarks
	_, err = tx.Exec("UPDATE gskbookmarks SET xhsum = xhash(printf('%s+%s+%s+%s',url,metadata,tags,desc))")
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
