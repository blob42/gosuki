// Copyright (c) 2023-2025 Chakib Ben Ziane <contact@blob42.xyz> and [`GoSuki` contributors]
// (https://github.com/blob42/gosuki/graphs/contributors).
//
// All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This file is part of GoSuki.
//
// GoSuki is free software: you can redistribute it and/or modify it under the terms of
// the GNU Affero General Public License as published by the Free Software Foundation,
// either version 3 of the License, or (at your option) any later version.
//
// GoSuki is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
// without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR
// PURPOSE.  See the GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License along with
// gosuki.  If not, see <http://www.gnu.org/licenses/>.

// Package database provides functionality for managing and synchronizing SQLite databases,
// specifically for bookmark data. It includes methods for syncing data between databases,
// caching, and disk persistence.
//
// The package supports:
//   - Syncing data from one database to another (upsert or update)
//   - Synchronizing in-memory cache databases to disk
//   - Copying entire databases
//   - Scheduling periodic sync operations
//   - Handling SQLite-specific constraints and errors
//
// The `SyncTo` method implements a manual UPSERT operation, which attempts to insert
// records from a source database to a destination database. If an insertion fails due
// to a constraint (e.g., duplicate URL), it will attempt to update the existing record.
//
// The `backupToDisk` method provides a way to sync a database to a specified disk path,
// using SQLite's backup API for efficient copying.
//
// The `SyncFromDisk` method allows for restoring data from a disk file into a database.
//
// The `CopyTo` method is used to copy an entire database from one location to another.
//
// The `SyncToCache` method is used to sync a database to an in-memory cache, either by
// copying the entire database or by performing a sync operation if the cache is not empty.
//
// This package also includes a scheduler for debounced sync operations to disk, which
// prevents excessive disk writes and ensures that syncs happen at regular intervals.
//
// The package uses the `sqlx` package for database operations and `log` for logging.
//
// See the individual function documentation for more details about their usage and behavior.
package database

import (
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	sqlite3 "github.com/mattn/go-sqlite3"
)

var mu sync.Mutex

// Manual UPSERT:
// For every row in `src` try to insert it into `dst`. if if fails then try to
// update it. It means `src` is synced to `dst`
func (src *DB) SyncTo(dst *DB) {
	var sqlite3Err sqlite3.Error
	var existingUrls []*RawBookmark

	log.Debugf("syncing <%s> to <%s>", src.Name, dst.Name)

	getSourceTable, err := src.Handle.Preparex(`SELECT * FROM gskbookmarks`)
	defer func() {
		err = getSourceTable.Close()
		if err != nil {
			log.Error(err)
		}
	}()
	if err != nil {
		log.Error(err)
	}

	getDstTags, err := dst.Handle.Preparex(
		`SELECT tags FROM gskbookmarks WHERE url=? LIMIT 1`,
	)

	defer func() {
		err = getDstTags.Close()

		if err != nil {
			log.Error(err)
		}
	}()

	if err != nil {
		log.Error(err)
	}

	tryInsertDstRow, err := dst.Handle.Preparex(
		`INSERT INTO
		gskbookmarks(url, metadata, tags, desc, flags, module)
		VALUES (?, ?, ?, ?, ?, ?)`,
	)
	defer func() {
		err = tryInsertDstRow.Close()
		if err != nil {
			log.Error(err)
		}
	}()

	if err != nil {
		log.Error(err)
	}

	updateDstRow, err := dst.Handle.Preparex(
		`UPDATE gskbookmarks
		SET (metadata, tags, desc, modified, flags) = (?,?,?,strftime('%s'),?)
		WHERE url=?
		`,
	)

	defer func() {
		err = updateDstRow.Close()
		if err != nil {
			log.Error(err)
		}
	}()

	if err != nil {
		log.Error(err)
	}

	srcTable, err := getSourceTable.Queryx()
	if err != nil {
		log.Error(err)
	}

	dstTx, err := dst.Handle.Beginx()
	if err != nil {
		log.Error(err)
	}

	// Start syncing all entries from source table
	for srcTable.Next() {

		// Fetch on row
		scan := RawBookmark{}
		err = srcTable.StructScan(&scan)
		if err != nil {
			log.Error(err)
		}

		// Try to insert to row in dst table
		_, err = dstTx.Stmtx(tryInsertDstRow).Exec(
			scan.URL,
			scan.Metadata,
			scan.Tags,
			scan.Desc,
			scan.Flags,
			scan.Module,
		)

		if err != nil {
			sqlite3Err = err.(sqlite3.Error)
		}

		if err != nil && sqlite3Err.Code != sqlite3.ErrConstraint {
			log.Error(err)
		}

		// Record already exists in dst table, we need to use update instead.
		if err != nil && sqlite3Err.Code == sqlite3.ErrConstraint {
			existingUrls = append(existingUrls, &scan)
		}
	}

	err = dstTx.Commit()
	if err != nil {
		log.Error(err)
	}

	// Start a new transaction to update the existing urls
	dstTx, err = dst.Handle.Beginx()
	if err != nil {
		log.Error(err)
	}

	// Traverse existing urls and try an update this time
	for _, scan := range existingUrls {
		var tags string

		//log.Debugf("updating existing %s", scan.Url)

		getDstTags.Get(&tags, scan.URL)

		srcTags := TagsFromString(scan.Tags, TagSep)

		dstTags := TagsFromString(tags, TagSep)

		tagMap := make(map[string]bool)
		for _, v := range srcTags.tags {
			tagMap[v] = true
		}
		for _, v := range dstTags.tags {
			tagMap[v] = true
		}

		newTags := &Tags{delim: TagSep} //merged tags
		for k := range tagMap {
			newTags.Add(k)
		}
		newTagsStr := newTags.StringWrap()

		_, err = dstTx.Stmtx(updateDstRow).Exec(
			scan.Metadata,
			newTagsStr,
			scan.Desc,
			0, //flags
			scan.URL,
		)

		if err != nil {
			log.Errorf("%s: %s", err, scan.URL)
		}
		log.Debugf("synced %s to %s", scan.URL, dst.Name)
	}

	err = dstTx.Commit()
	if err != nil {
		log.Error(err)
	}

	// If we are syncing to memcache, sync cache to disk
	if dst.Name == CacheName {
		ScheduleBackupToDisk()
	}
}

var syncQueue = make(chan any)

// Sync all databases to disk in a goroutine using a debouncer
func cacheSyncScheduler(input <-chan any) {
	log.Debug("starting cache sync scheduler")

	queue := make(chan any, 100)

	// debounce interval
	timer := time.NewTimer(0)
	for {
		select {
		case <-input:
			// log.Debug("debouncing sync to disk")
			timer.Reset(dbConfig.SyncInterval)
			// log.Debugf("sync que len is %d", len(queue))
			select {
			case queue <- true:
				// Writing to queue will not block
				// log.Debug("pushed sync to disk to queue")
			default:
				log.Debug("queue is full, dropping sync to disk request")
				continue
			}
		case <-timer.C:
			if len(queue) > 0 {
				if Cache.DB == nil {
					log.Fatalf("cache db is nil")
				}
				if err := Cache.SyncToDisk(GetDBFullPath()); err != nil {
					log.Fatalf("failed to sync cache to disk: %s", err)
				}

				// empty the queue
				for len(queue) > 0 {
					<-queue
				}
			}
		}
	}
}

// TODO: add `force` param to force sync
func ScheduleBackupToDisk() {
	go func() {
		log.Debug("received sync to disk request")
		syncQueue <- true
	}()
}

func startSyncScheduler() {
	go cacheSyncScheduler(syncQueue)
}

// backupToDisk copies the `src` database contents to a file on disk.
// It creates a backup of the source database (src) to the specified dbpath.
// The function is safe for concurrent use as it acquires a mutex.
// Returns an error if any step fails, including database connection issues,
// backup execution errors, or invalid configuration.
// Uses SQLite's backup API via the sqlx package, requiring the driver to support it.
func (src *DB) backupToDisk(dbpath string) error {
	log.Debugf("copying <%s> to <%s>", src.Name, dbpath)
	defer func() {
		if err := recover(); err != nil {
			log.Error("recovered in backupToDisk", err)
		}
	}()

	mu.Lock()
	defer mu.Unlock()

	//log.Debugf("[flush] openeing <%s>", src.path)
	srcDB, err := sqlx.Open(DriverBackupMode, src.Path)
	defer flushSqliteCon(srcDB)
	if err != nil {
		return err
	}
	if err = srcDB.Ping(); err != nil {
		return err
	}

	//log.Debugf("[flush] opening <%s>", DB_FILENAME)

	dbURI := fmt.Sprintf("file:%s", dbpath)
	bkDB, err := sqlx.Open(DriverBackupMode, dbURI)
	defer flushSqliteCon(bkDB)
	if err != nil {
		return err
	}

	err = bkDB.Ping()
	if err != nil {
		return err
	}

	if len(_sql3BackupConns) < 2 {
		panic("not enough sql connections for backup call")
	}

	if _sql3BackupConns[0] == nil {
		log.Error("nil sql connection")
		return fmt.Errorf("nil sql connection")
	}

	bkp, err := _sql3BackupConns[1].Backup("main", _sql3BackupConns[0], "main")
	if err != nil {
		return err
	}

	_, err = bkp.Step(-1)
	if err != nil {
		return err
	}

	bkp.Finish()
	log.Infof("copied <%s> to <%s>", src.Name, dbpath)

	return nil
}

func (dst *DB) SyncFromDisk(dbpath string) error {

	log.Debugf("syncing <%s> to <%s>", dbpath, dst.Name)

	dbURI := fmt.Sprintf("file:%s", dbpath)
	srcDB, err := sqlx.Open(DriverBackupMode, dbURI)
	defer flushSqliteCon(srcDB)
	if err != nil {
		return err
	}
	srcDB.Ping()

	//log.Debugf("[flush] opening <%s>", DB_FILENAME)
	bkDB, err := sqlx.Open(DriverBackupMode, dst.Path)
	defer flushSqliteCon(bkDB)
	if err != nil {
		return err
	}
	bkDB.Ping()

	bk, err := _sql3BackupConns[1].Backup("main", _sql3BackupConns[0], "main")
	if err != nil {
		return err
	}

	_, err = bk.Step(-1)
	if err != nil {
		return err
	}

	bk.Finish()

	return nil
}

// Copy from src DB to dst DB using sqlite3 backup mode
// `dst` is overwritten
func (src *DB) CopyTo(dst *DB, dstName, srcName string) {

	log.Debugf("copying <%s> to <%s>", src.Name, dst.Name)
	mu.Lock()
	defer mu.Unlock()

	srcDB, err := sqlx.Open(DriverBackupMode, src.Path)
	defer func() {
		srcDB.Close()
		_sql3BackupConns = _sql3BackupConns[:len(_sql3BackupConns)-1]
	}()
	if err != nil {
		log.Error(err)
	}

	srcDB.Ping()

	dstDB, err := sqlx.Open(DriverBackupMode, dst.Path)
	defer func() {
		dstDB.Close()
		_sql3BackupConns = _sql3BackupConns[:len(_sql3BackupConns)-1]
	}()
	if err != nil {
		log.Error(err)
	}
	dstDB.Ping()

	bk, err := _sql3BackupConns[1].Backup("main", _sql3BackupConns[0], "main")
	if err != nil {
		log.Error(err)
	}

	_, err = bk.Step(-1)
	if err != nil {
		log.Error(err)
	}

	bk.Finish()
}

func (src *DB) SyncToCache() error {
	Cache.mu.Lock()
	defer Cache.mu.Unlock()
	if Cache.DB == nil {
		return fmt.Errorf("cache db is nil")
	}

	empty, err := Cache.IsEmpty()

	// otherwise move on and check if error is table does not exist
	sql3err, isSQL3Err := err.(sqlite3.Error)
	if err != nil && sql3err.Code != sqlite3.ErrError {
		return fmt.Errorf("error checking if cache is empty: %w", err)
	}
	if empty || (isSQL3Err && sql3err.Code == sqlite3.ErrError) {
		log.Debugf("cache is empty, copying <%s> to <%s>", src.Name, CacheName)
		src.CopyTo(Cache.DB, "main", "main")
	} else {
		log.Debugf("syncing <%s> to cache", src.Name)
		src.SyncTo(Cache.DB)
	}
	return nil
}
