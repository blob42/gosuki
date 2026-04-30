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

// Performs the database schema migration from version 3 to version 4.
// This migration adds a composite index on gskbookmarks(version, node_id) to
// accelerate P2P sync change detection queries that scan for bookmarks
// modified after a given version and originating from other nodes:
//
//	SELECT * FROM gskbookmarks WHERE version > ? AND hex(node_id) IS NOT hex(?)
//
// The composite index enables an index range scan on `version` with
// in-index filtering on `node_id`, avoiding a full table scan.
func (db *DB) migrateToVersion4() error {
	log.Debug("DB schema: migrating to v4")
	tx, err := db.Handle.Begin()
	if err != nil {
		return DBError{DBName: db.Name, Err: err}
	}

	_, err = tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_gskbookmarks_version_node_id
			ON gskbookmarks(version, node_id)
	`)
	if err != nil {
		tx.Rollback()
		return DBError{DBName: db.Name, Err: err}
	}

	if err := tx.Commit(); err != nil {
		return DBError{DBName: db.Name, Err: err}
	}

	return nil
}
