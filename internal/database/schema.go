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
	"context"
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/blob42/gosuki/internal/utils"
)

/* Schema Versions:
  - Version 1: Initial version
  - Version 2:
	  - Altered gskbookmarks table with:
	  - Added column xhsum
	  - Restored column id as primary key
	  - Restored URL unique constraint
  - Version 3: Added synchronization capabilities:
	  - Added version column to gskbookmarks table
	  - Added node_id column to gskbookmarks table
	  - Created sync_nodes table for node synchronization management
  - Version 4: Added performance indexes:
	  - Created idx_gskbookmarks_version_node_id composite index
	    on gskbookmarks(version, node_id) for P2P sync change detection
*/

const CurrentSchemaVersion = 4

const (

	// metadata: name or title of resource
	// modified: time.Now().Unix()
	// desc:
	// flags: designed to be extended in future using bitwise masks
	// Masks:
	//     0b00000001: set title immutable ((do not change title when updating the bookmarks from the web ))
	QCreateSchema = `
    CREATE TABLE IF NOT EXISTS gskbookmarks (
		id INTEGER PRIMARY KEY,
		URL TEXT NOT NULL UNIQUE,
		metadata TEXT default '',
		tags TEXT default '',
		desc TEXT default '',
		modified INTEGER DEFAULT (strftime('%s')),
		flags INTEGER DEFAULT 0,
		module TEXT DEFAULT '' ,
		xhsum TEXT DEFAULT '',
		version INTEGER DEFAULT 0,
		node_id BLOB
	);

	CREATE TABLE IF NOT EXISTS sync_nodes (
		ordinal INTEGER PRIMARY KEY,
		node_id BLOB NOT NULL UNIQUE,
		version INTEGER NOT NULL
	)
	`

	// The following view and and triggers provide buku compatibility
	QCreateView = `CREATE VIEW bookmarks AS
	SELECT id, URL, metadata, tags, desc, flags
	FROM gskbookmarks`

	QCreateInsertTrigger = `CREATE TRIGGER bookmarks_insert
	INSTEAD OF INSERT ON bookmarks
	BEGIN
		INSERT INTO gskbookmarks (URL, metadata, tags, desc, modified, flags, module)
		VALUES (
			new.URL,
			COALESCE(new.metadata, ''),
			COALESCE(new.tags, ''),
			COALESCE(new.desc, ''),
			strftime('%s'),
			COALESCE(new.flags, 0),
			'buku'
		);
	END
	`

	QCreateUpdateTrigger = `
	CREATE TRIGGER bookmarks_update
	INSTEAD OF UPDATE ON bookmarks
	BEGIN
		UPDATE gskbookmarks
		SET
			URL = COALESCE(new.URL, (SELECT URL FROM gskbookmarks WHERE id = old.id)),
			metadata = COALESCE(new.metadata, (SELECT metadata FROM gskbookmarks WHERE id = old.id)),
			tags = COALESCE(new.tags, (SELECT tags FROM gskbookmarks WHERE id = old.id)),
			desc = COALESCE(new.desc, (SELECT desc FROM gskbookmarks WHERE id = old.id)),
			modified = strftime('%s'),
			flags = COALESCE(new.flags, (SELECT flags FROM gskbookmarks WHERE id = old.id))
		WHERE id = old.id;
	END
	`

	QCreateSchemaVersion = `
		CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER PRIMARY KEY
		)
	`
)

// backupDB creates a copy of the database file before migration.
// The backup is named gosuki-v<version>.db.bak in the same directory as the database.
// Skips backup for in-memory databases.
func backupDB(db *DB, version int) error {
	if db.Type == DBTypeInMemory {
		return nil
	}

	backupName := fmt.Sprintf("gosuki-v%d.db.bak", version)
	backupPath := filepath.Join(filepath.Dir(db.filePath), backupName)

	if err := utils.CopyFileToDst(db.filePath, backupPath); err != nil {
		return DBError{DBName: db.Name, Err: fmt.Errorf("backing up database: %w", err)}
	}

	log.Info("database backup created", "path", backupPath, "version", version)
	return nil
}

func checkDBVersion(db *DB) error {
	var err error
	var version int
	var tableExists bool
	log.Debug("checking schema version")

	// only apply checks and migrations to db on disk
	if db.Name != "gosuki_db" {
		return nil
	}

	// Create schema_version table if not exists
	_, err = db.Handle.Exec(QCreateSchemaVersion)
	if err != nil {
		return DBError{DBName: db.Name, Err: err}
	}

	// initial version
	err = db.Handle.QueryRow("SELECT version FROM schema_version").Scan(&version)
	if err == sql.ErrNoRows {
		log.Debug("unversioned schema detected")
		_, err = db.Handle.Exec("INSERT INTO schema_version (version) VALUES (1)")
		if err != nil {
			return DBError{DBName: db.Name, Err: err}
		}

		log.Debug("created schema_version table")

		// checking if ondisk db needs to be upgraded to version 1
		err = db.Handle.QueryRowx(`
			SELECT 1 FROM sqlite_master 
			WHERE type='table' AND name='bookmarks'
			`).Scan(&tableExists)

		if err != nil && err != sql.ErrNoRows {
			return DBError{DBName: db.Name, Err: err}
		}

		// Upgrade ondisk gosuki db to version 1 with schema tracking
		if tableExists {
			log.Debug("bookmarks table exists: migrating to v1")

			tx, err := db.Handle.Begin()
			if err != nil {
				return DBError{DBName: db.Name, Err: err}
			}

			log.Debug("creating gskbookmarks table")
			if _, err = tx.Exec(QCreateSchema); err != nil {
				tx.Rollback()
				return DBError{DBName: db.Name, Err: err}
			}

			log.Debug("moving table bookmarks to gskbookmarks")
			if _, err := tx.Exec(`
				INSERT INTO gskbookmarks
					(URL, metadata, tags, desc, modified, flags, module)
					SELECT URL, metadata, tags, desc, modified, flags, module
					FROM bookmarks`); err != nil {
				tx.Rollback()
				return DBError{DBName: db.Name, Err: err}
			}

			log.Debug("dropping `bookmarks` table")
			if _, err := tx.Exec("DROP TABLE bookmarks"); err != nil {
				tx.Rollback()
				return DBError{DBName: db.Name, Err: err}
			}

			log.Debug("creating `bookmarks` view")
			if _, err = tx.Exec(QCreateView); err != nil {
				tx.Rollback()
				return DBError{DBName: db.Name, Err: err}
			}

			if _, err = tx.Exec(QCreateInsertTrigger); err != nil {
				tx.Rollback()
				return DBError{DBName: db.Name, Err: err}
			}

			if _, err = tx.Exec(QCreateUpdateTrigger); err != nil {
				tx.Rollback()
				return DBError{DBName: db.Name, Err: err}
			}

			if err = tx.Commit(); err != nil {
				return DBError{DBName: db.Name, Err: err}
			}
		}

		version = 1
	}

	if version > CurrentSchemaVersion {
		return fmt.Errorf("unrecognized db version %d: current=%d", version, CurrentSchemaVersion)
	}

	if version < CurrentSchemaVersion {
		if err = backupDB(db, version); err != nil {
			return err
		}

		for version < CurrentSchemaVersion {
			switch version {
			case 1:
				if err = db.migrateToVersion2(); err != nil {
					return err
				}
				version = 2
			case 2:
				if err = db.migrateToVersion3(); err != nil {
					return err
				}
				version = 3
			case 3:
				if err = db.migrateToVersion4(); err != nil {
					return err
				}
				version = 4
			}
		}
	}

	// Update the version in the schema_version table
	_, err = db.Handle.Exec("UPDATE schema_version SET version = ?", version)
	if err != nil {
		return DBError{DBName: db.Name, Err: err}
	}

	log.Debug("schema", "version", version)

	return err
}

func (db *DB) InitSchema(ctx context.Context) error {

	// Populate db schema
	tx, err := db.Handle.Begin()
	if err != nil {
		return DBError{DBName: db.Name, Err: err}
	}

	if _, err = tx.ExecContext(ctx, QCreateSchema); err != nil {
		tx.Rollback()
		return DBError{DBName: db.Name, Err: err}
	}

	if _, err = tx.ExecContext(ctx, QCreateView); err != nil {
		tx.Rollback()
		return DBError{DBName: db.Name, Err: err}
	}

	if _, err = tx.ExecContext(ctx, QCreateInsertTrigger); err != nil {
		tx.Rollback()
		return DBError{DBName: db.Name, Err: err}
	}

	if _, err = tx.ExecContext(ctx, QCreateUpdateTrigger); err != nil {
		tx.Rollback()
		return DBError{DBName: db.Name, Err: err}
	}

	if err = tx.Commit(); err != nil {
		return DBError{DBName: db.Name, Err: err}
	}

	err = checkDBVersion(db)
	if err != nil {
		return fmt.Errorf("checking schema version: %w", err)
	}

	log.Debugf("<%s> initialized", db.Name)

	return nil
}
