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

/*
Package database provides a comprehensive suite of tools for managing, synchronizing, and persisting SQLite databases
specifically tailored for bookmark data. It implements efficient data synchronization between in-memory caches and
disk-based databases, handles SQLite-specific operations, and includes scheduling mechanisms for optimized disk access.

Key Features:
- Bidirectional synchronization between databases (upsert/update operations)
- In-memory cache management with automatic disk persistence
- Full database copying and restoration capabilities
- Scheduled debounced sync operations to prevent excessive I/O
- Robust error handling for SQLite constraints and operations
- Utilizes SQLite's native backup API for efficient data transfers

The package is designed to support a two-level caching architecture:
1. L1 (in-memory cache) - for fast access and temporary storage
2. L2 (disk-based cache) - for persistent storage and data integrity

It provides methods for:
- Syncing data between databases (SyncTo, SyncFromDisk)
- Copying entire databases (CopyTo)
- Managing cache-to-disk synchronization (SyncToCache, backupToDisk)
- Scheduling periodic sync operations (ScheduleBackupToDisk)

The package leverages the sqlx library for database operations.
All database operations should be thread-safe through the use of mutexes and proper transaction management.

Note: This package requires proper initialization of database connections and configuration parameters
(e.g., sync intervals, database paths) before use.

See individual function documentation for specific usage patterns and behavior details.
*/

package database

// TODO: add context to all queries

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	sqlite3 "github.com/mattn/go-sqlite3"
)

var mu sync.Mutex

/*
	SyncTo synchronizes bookmarks from the source DB (src) to the destination DB (dst).

It performs the following steps:
 1. Reads all entries from src's gskbookmarks table
 2. Attempts to insert each entry into dst's gskbookmarks table
 3. For existing entries (due to URL constraints), captures their hashes and
    processes them in a second transaction for potential updates
 4. Updates existing entries only if there are changes in metadata, tags, or description
 5. Commits transactions for both insert and update phases
 6. If dst is a memcache, schedules a disk backup after completion

The synchronization uses SQLite transactions for consistency and handles
duplicate URL constraints by comparing hash values. Tags are merged and
normalized during updates.
*/
func (src *DB) SyncTo(dst *DB) {
	var sqlite3Err sqlite3.Error
	var existingUrls = make(map[uint64]*RawBookmark)

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

	tryInsertDstRow, err := dst.Handle.Preparex(
		`INSERT INTO
		gskbookmarks(
			url,
			metadata,
			tags,
			desc,
			flags,
			module,
			xhsum
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		log.Error("prepare stmt", "err", err)
	}

	defer func() {
		err = tryInsertDstRow.Close()
		if err != nil {
			log.Error("closing statement: ", "err", err)
		}
	}()

	updateDstRow, err := dst.Handle.Preparex(
		`UPDATE gskbookmarks
		SET (
			metadata,
			tags,
			desc,
			modified,
			flags,
			module,
			xhsum
		) = (
			CASE WHEN ? != '' THEN ? ELSE metadata END,
			?,
			CASE WHEN ? != '' THEN ? ELSE desc END,
			strftime('%s'),
			?,
			?,
			?
		)
		WHERE url=? 
		`,
	)
	if err != nil {
		log.Error("closing statement: ", "err", err)
	}

	defer func() {
		err = updateDstRow.Close()
		if err != nil {
			log.Error(err)
		}
	}()

	srcTable, err := getSourceTable.Queryx()
	if err != nil {
		log.Error("get src table: ", "err", err)
	}

	dstTx, err := dst.Handle.Beginx()
	if err != nil {
		log.Error("begin tx", "err", err)
		dstTx.Rollback()
		return
	}

	getDstTagsStmt, err := dst.Handle.Preparex(
		`SELECT tags FROM gskbookmarks WHERE url=? LIMIT 1`,
	)

	// Start syncing all entries from source table
	for srcTable.Next() {

		// Fetch on row
		scan := RawBookmark{}
		err = srcTable.StructScan(&scan)
		if err != nil {
			log.Error("scan", "err", err)
			continue
		}

		// Try to insert to row in dst table
		_, err = dstTx.Stmtx(tryInsertDstRow).Exec(
			scan.URL,
			scan.Metadata,
			scan.Tags,
			scan.Desc,
			scan.Flags,
			scan.Module,
			xhsum(
				scan.URL,
				scan.Metadata,
				scan.Tags,
				scan.Desc,
			),
		)

		if err != nil {
			sqlite3Err = err.(sqlite3.Error)
		}

		if err != nil && sqlite3Err.Code != sqlite3.ErrConstraint {
			log.Error("inserting", "err", err)
			continue
		}

		// Record already existing bookmarks in `dst` then proceed to UPDATE.
		if err != nil && sqlite3Err.Code == sqlite3.ErrConstraint {

			// check original hash of bookmark
			var srcBkHash xxhashsum
			err = dstTx.QueryRowx("SELECT xhsum FROM gskbookmarks WHERE url = ?", scan.URL).Scan(&srcBkHash)
			if err != nil {
				log.Error("select xhsum from", "src", L2Cache.Name, "url", scan.URL, "err", err)
				continue

			}

			existingUrls[uint64(srcBkHash)] = &scan
		}
	}

	err = dstTx.Commit()
	if err != nil {
		log.Error("rolling back after error", "err", err)
		dstTx.Rollback()
	}

	dstTx, err = dst.Handle.Beginx()
	if err != nil {
		log.Error("begin tx", "err", err)
	}

	// Loop performing the update for each existing bookmark
	for hash, scan := range existingUrls {
		var tags string
		//log.Debugf("updating existing %s", scan.Url)

		if err = dstTx.Stmtx(getDstTagsStmt).Get(&tags, scan.URL); err != nil {
			log.Error("get tags query", "err", err)
		}

		srcTags := TagsFromString(scan.Tags, TagSep).Sort()
		dstTags := TagsFromString(tags, TagSep).Sort()

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
		newTagsStr := newTags.Sort().StringWrap()
		newHash := xhsum(scan.URL, scan.Metadata, newTagsStr, scan.Desc)

		if strconv.FormatUint(hash, 10) == newHash {
			continue
		}

		_, err = dstTx.Stmtx(updateDstRow).Exec(
			scan.Metadata,
			scan.Metadata,
			newTagsStr,
			scan.Desc,
			scan.Desc,
			0, //flags
			scan.Module,
			newHash,
			scan.URL,
		)

		if err != nil {
			log.Errorf("%s: %s", err, scan.URL)
		}
		log.Debugf("synced %s to %s", scan.URL, dst.Name)
	}

	err = dstTx.Commit()
	if err != nil {
		dstTx.Rollback()
		log.Error("sync:commit", "err", err)
	}

	// If we are syncing to memcache, schedule a write to disk
	if dst.Name == CacheName {
		ScheduleBackupToDisk()
	}
}

var syncQueue = make(chan any)

// cacheSyncScheduler starts a scheduler that debounces cache sync operations to
// disk. it uses a two-level caching strategy: first syncing the main cache to
// an l2 cache, then backing up the l2 cache to disk. the scheduler processes
// input events to trigger syncs, with a debounce interval defined by
// dbconfig.syncinterval. if the internal queue is full, incoming sync requests
// are dropped. on timer expiration, it performs the sync and backup operations.
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
				// Backup in 2 levels
				// 1. Sync Cache to L2 cache
				// 2. Backup L2 cache to disk
				// This allows comparing bookmark change checksums against the
				// disk database. In other words, L1 cache used for efficiency
				// and L2 ensures data integrity and avoids unecessary I/O.
				Cache.SyncTo(L2Cache.DB)
				if err := L2Cache.backupToDisk(GetDBFullPath()); err != nil {
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
