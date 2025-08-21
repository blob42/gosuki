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

The package leverages the sqlx library for database operations. All database
operations should be thread-safe through the use of mutexes and proper
transaction management.

Note: This package requires proper initialization of database connections and
configuration parameters (e.g., sync intervals, database paths) before use.

See individual function documentation for specific usage patterns and behavior
details.
*/

package database


import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jmoiron/sqlx"
	sqlite3 "github.com/mattn/go-sqlite3"

	"github.com/blob42/gosuki"
	"github.com/blob42/gosuki/hooks"
	"github.com/blob42/gosuki/pkg/config"
)

var (
	diskDBmu    sync.Mutex
	cacheMu     sync.Mutex
	SyncTrigger = atomic.Bool{}
)

/*
SyncTo synchronizes bookmarks from the source database to the destination
database using the current Lamport clock value.

This function performs local-only synchronization by:
1. Reading all entries from the source database's gskbookmarks table
2. Attempting to insert each entry into the destination database's gskbookmarks
table
3. Handling duplicate URL constraints by comparing hash values and updating
existing entries
4. Maintaining versioning through Lamport clock synchronization
5. Scheduling disk backup when syncing to memcache

It is designed for local synchronization scenarios where causal ordering needs
to be maintained using Lamport timestamps that can be used for multi-device
synchronization.

Example usage:

	srcDB := NewDB("source.db")
	dstDB := NewDB("destination.db")
	srcDB.SyncTo(dstDB)
*/
func (src *DB) SyncTo(dst *DB) {
	src.SyncToClock(dst, Clock.Value)
}

/*
SyncToClock synchronizes bookmarks from the source DB (src) to the destination
DB (dst) using Lamport clock synchronization for peer-to-peer consistency.

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
normalized during updates. Lamport clock is used to maintain versioning
and ensure proper ordering of operations in distributed systems.

Parameters:
  - dst: The destination database to sync bookmarks to
  - clock: The Lamport clock value to use for versioning

Behavior:
- When syncing to L2 cache, increments the version field on successful inserts
- Merges tags from both source and destination when updating existing entries
- Normalizes merged tags by sorting them alphabetically
- Only updates entries when there are actual changes in metadata, tags, or
description
- Schedules disk backup when syncing to memcache (CacheName)
- Uses Lamport clock for p2p synchronization to maintain causal ordering
*/
func (src *DB) SyncToClock(dst *DB, remoteClock uint64) {
	var err error
	var sqlite3Err sqlite3.Error
	var isSqlErr bool
	var existingUrls = make(map[uint64]*RawBookmark)

	log.Debugf("syncing <%s> to <%s>", src.Name, dst.Name)
	cacheMu.Lock()
	defer cacheMu.Unlock()

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
			xhsum,
			version,
			node_id
			
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
			xhsum,
			version,
			node_id
		) = (
			CASE WHEN ? != '' THEN ? ELSE metadata END,
			?,
			CASE WHEN ? != '' THEN ? ELSE desc END,
			strftime('%s'),
			?,
			?,
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
			remoteClock,
			scan.NodeID,
		)

		isSqlErr = false
		if err != nil {
			sqlite3Err, isSqlErr = err.(sqlite3.Error)
		}

		if isSqlErr && sqlite3Err.Code != sqlite3.ErrConstraint {
			log.Error("inserting", "err", err)
			continue
		}

		// Record already existing bookmarks in `dst` then proceed to UPDATE.
		if isSqlErr && sqlite3Err.Code == sqlite3.ErrConstraint {

			// check original hash of bookmark
			var oldBkHash xxhashsum
			err = dstTx.QueryRowx("SELECT xhsum FROM gskbookmarks WHERE url = ?", scan.URL).Scan(&oldBkHash)
			if err != nil {
				log.Error("select xhsum from", "src", L2Cache.Name, "url", scan.URL, "err", err)
				continue

			}

			existingUrls[uint64(oldBkHash)] = &scan

			// insertion success on l2 cache, update clock
		} else if err == nil && dst.Name == L2CacheName {
			log.Trace("inserted", "url", scan.URL, "tags", scan.Tags)
			hooksQueue <- hooks.HookJob{
				Book: scan.AsBookmark(),
				Kind: hooks.GlobalInsertHook,
			}

			_, err = dstTx.Exec("UPDATE gskbookmarks SET version = ? WHERE URL = ?",
				Clock.Tick(remoteClock), scan.URL)
			if err != nil {
				log.Error("insert:clock-inc", "err", err)
				dstTx.Rollback()
			}
		}
	}

	err = dstTx.Commit()
	if err != nil {
		log.Error("sync", "from", src.Name, "to", dst.Name, "err", err)
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

		srcTags := tagsFromString(scan.Tags, TagSep).Sort()
		dstTags := tagsFromString(tags, TagSep).Sort()

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

		clock := remoteClock

		if dst.Name == L2CacheName {
			clock = Clock.Tick(remoteClock)
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
			clock,
			scan.NodeID,
			scan.URL,
		)

		if err != nil {
			log.Errorf("%s: %s", err, scan.URL)
			// update success
		} else {
			log.Trace("updated", "url", scan.URL, "tags", newTagsStr)
			hooksQueue <- hooks.HookJob{
				Book: &gosuki.Bookmark{
					URL:    scan.URL,
					Title:  scan.Metadata,
					Tags:   newTags.tags,
					Desc:   scan.Desc,
					Module: scan.Module,
				},
				Kind: hooks.GlobalUpdateHook,
			}
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

var (
	syncQueue  chan any
	hooksQueue chan hooks.HookJob
)

// cacheSyncScheduler starts a scheduler that debounces cache sync operations to
// disk. it uses a two-level caching strategy: first syncing the main cache to
// an l2 cache, then backing up the l2 cache to disk. the scheduler processes
// input events to trigger syncs, with a debounce interval defined by
// dbconfig.syncinterval. if the internal queue is full, incoming sync requests
// are dropped. on timer expiration, it performs the sync and backup operations.
func cacheSyncScheduler(input <-chan any) {
	log.Debug("starting cache sync scheduler")

	queue := make(chan any, 100)
	defer close(queue)

	// debounce interval
	timer := time.NewTimer(0)
	for {
		select {
		case <-input:
			// log.Debug("debouncing sync to disk")
			timer.Reset(Config.SyncInterval)
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
				if err := L2Cache.BackupToDisk(config.DBPath); err != nil {
					log.Fatalf("failed to sync l2 cache to disk: %s", err)
				} else {
					SyncTrigger.Store(true)
				}

				// empty the queue
				for len(queue) > 0 {
					<-queue
				}
			}
		}
	}
}

func ScheduleBackupToDisk() {
	go func() {
		log.Debug("received sync to disk request")
		syncQueue <- true
	}()
}

func startSchedulers() {
	syncQueue = make(chan any)
	hooksQueue = make(chan hooks.HookJob, 100)
	go cacheSyncScheduler(syncQueue)
	go hooks.HooksScheduler(hooksQueue)
}

// BackupToDisk copies the `src` database contents to a file on disk.
// It creates a backup of the source database (src) to the specified dbpath.
// The function is safe for concurrent use as it acquires a mutex.
// Returns an error if any step fails, including database connection issues,
// backup execution errors, or invalid configuration.
// Uses SQLite's backup API via the sqlx package, requiring the driver to support it.
func (src *DB) BackupToDisk(dbpath string) error {
	log.Debugf("copying <%s> to <%s>", src.Name, dbpath)
	defer func() {
		if err := recover(); err != nil {
			log.Error("recovered in backupToDisk", err)
		}
	}()

	diskDBmu.Lock()
	defer diskDBmu.Unlock()

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

	if err = bkDB.Ping(); err != nil {
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
	if err = srcDB.Ping(); err != nil {
		return err
	}

	//log.Debugf("[flush] opening <%s>", DB_FILENAME)
	bkDB, err := sqlx.Open(DriverBackupMode, dst.Path)
	defer flushSqliteCon(bkDB)
	if err != nil {
		return err
	}

	if err = bkDB.Ping(); err != nil {
		return err
	}

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
	diskDBmu.Lock()
	defer diskDBmu.Unlock()

	srcDB, err := sqlx.Open(DriverBackupMode, src.Path)
	defer func() {
		srcDB.Close()
		_sql3BackupConns = _sql3BackupConns[:len(_sql3BackupConns)-1]
	}()
	if err != nil {
		log.Error(err)
	}

	if err = srcDB.Ping(); err != nil {
		log.Error(err)
	}

	dstDB, err := sqlx.Open(DriverBackupMode, dst.Path)
	defer func() {
		dstDB.Close()
		_sql3BackupConns = _sql3BackupConns[:len(_sql3BackupConns)-1]
	}()
	if err != nil {
		log.Error(err)
	}
	if err = dstDB.Ping(); err != nil {
		log.Error(err)
	}

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
