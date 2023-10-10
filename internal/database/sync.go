//
// Copyright ⓒ 2023 Chakib Ben Ziane <contact@blob42.xyz> and [`GoSuki` contributors]
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
package database
// TODO: add context to all queries

import (
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	sqlite3 "github.com/mattn/go-sqlite3"

	"github.com/blob42/gosuki/pkg/config"
)

var mu sync.Mutex

// Manual UPSERT:
// For every row in `src` try to insert it into `dst`. if if fails then try to
// update it. It means `src` is synced to `dst`
func (src *DB) SyncTo(dst *DB) {
	var sqlite3Err sqlite3.Error
	var existingUrls []*SBookmark

	log.Debugf("syncing <%s> to <%s>", src.Name, dst.Name)

	getSourceTable, err := src.Handle.Prepare(`SELECT * FROM bookmarks`)
	defer func() {
		err = getSourceTable.Close()
		if err != nil {
			log.Critical(err)
		}
	}()
	if err != nil {
		log.Error(err)
	}

	getDstTags, err := dst.Handle.Prepare(
		`SELECT tags FROM bookmarks WHERE url=? LIMIT 1`,
	)

	defer func() {
		err = getDstTags.Close()

		if err != nil {
			log.Critical(err)
		}
	}()

	if err != nil {
		log.Error(err)
	}

	tryInsertDstRow, err := dst.Handle.Prepare(
		`INSERT INTO
		bookmarks(url, metadata, tags, desc, flags)
		VALUES (?, ?, ?, ?, ?)`,
	)
	defer func() {
		err = tryInsertDstRow.Close()
		if err != nil {
			log.Critical(err)
		}
	}()

	if err != nil {
		log.Error(err)
	}

	updateDstRow, err := dst.Handle.Prepare(
		`UPDATE bookmarks
		SET (metadata, tags, desc, modified, flags) = (?,?,?,strftime('%s'),?)
		WHERE url=?
		`,
	)

    defer func(){
        err = updateDstRow.Close()
        if err != nil {
            log.Critical()
        }
    }()
    
	if err != nil {
		log.Error(err)
	}

	srcTable, err := getSourceTable.Query()
	if err != nil {
		log.Error(err)
	}

	log.Debugf("starting transaction")
	dstTx, err := dst.Handle.Begin()
	if err != nil {
		log.Error(err)
	}


	// Start syncing all entries from source table
	log.Debugf("scanning entries in source table")
	for srcTable.Next() {

		// Fetch on row
		scan, err := ScanBookmarkRow(srcTable)
		if err != nil {
			log.Error(err)
		}

		// Try to insert to row in dst table
		_, err = dstTx.Stmt(tryInsertDstRow).Exec(
			scan.URL,
			scan.metadata,
			scan.tags,
			scan.desc,
			scan.flags,
		)

		if err != nil {
			sqlite3Err = err.(sqlite3.Error)
		}

		if err != nil && sqlite3Err.Code != sqlite3.ErrConstraint {
			log.Error(err)
		}

		// Record already exists in dst table, we need to use update
		// instead.
		if err != nil && sqlite3Err.Code == sqlite3.ErrConstraint {
			existingUrls = append(existingUrls, scan)
		}
	}

	err = dstTx.Commit()
	if err != nil {
		log.Error(err)
	}

	// Start a new transaction to update the existing urls
	dstTx, err = dst.Handle.Begin() 
	if err != nil {
		log.Error(err)
	}

	// Traverse existing urls and try an update this time
	for _, scan := range existingUrls {
		var tags string

		//log.Debugf("updating existing %s", scan.Url)

		row := getDstTags.QueryRow(
			scan.URL,
		)
		row.Scan(&tags)

		//log.Debugf("src tags: %v", scan.tags)
		//log.Debugf("dst tags: %v", dstTags)
		srcTags := TagsFromString(scan.tags, TagSep)

		dstTags := TagsFromString(tags, TagSep)

		tagMap := make(map[string]bool)
		for _, v := range srcTags.tags {
			tagMap[v] = true
		}
		for _, v := range dstTags.tags {
			tagMap[v] = true
		}

		newTags := &Tags{delim: TagSep}//merged tags
		for k := range tagMap {
			newTags.Add(k)
		}
		newTagsStr := newTags.StringWrap()

		_, err = dstTx.Stmt(updateDstRow).Exec(
			scan.metadata,
			newTagsStr,
			scan.desc,
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
		err = dst.SyncToDisk(GetDBFullPath())
		if err != nil {
			log.Error(err)
		}
	}
}

var syncQueue = make(chan interface{})

// Sync all databases to disk in a goroutine using a debouncer
//TODO: add `force` param to force sync
func cacheSyncScheduler(input <-chan interface{}) {
	log.Debug("starting cache sync scheduler")

	// debounce interval
	queue := make(chan<- interface{}, 100)
	interval := 4 * time.Second
	timer := time.NewTimer(0)
	for {
		select {
		case <-input:
			log.Debug("debouncing sync to disk")
			timer.Reset(interval)
			select {
			case queue <- true:
				// Writing to queue would not block
			default:
				log.Critical("cache sync queue is full, something is wrong")
			}
		case <-timer.C:
			if len(queue) > 0 {
				log.Debug("syncing cache to disk")
				if Cache.DB == nil {
					log.Fatalf("cache db is nil")
				}
				if err := Cache.DB.SyncToDisk(GetDBFullPath()); err != nil {
					log.Fatalf("failed to sync cache to disk: %s", err)
				}
				queue = make(chan<- interface{})
			}
		}
	}
}

func ScheduleSyncToDisk() {
	go func() {
		log.Debug("received sync to disk request")
		syncQueue <- true
	}()
}

func StartSyncScheduler() {
	go cacheSyncScheduler(syncQueue)
}

func (src *DB) SyncToDisk(dbpath string) error {
	log.Debugf("Syncing <%s> to <%s>", src.Name, dbpath)
	defer func() {
		if err := recover(); err != nil {
			log.Critical("Recovered in SyncToDisk", err)
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

	if len(_sql3conns) < 2 {
		return fmt.Errorf("not enough sql connections for backup call")
	}

	if _sql3conns[0] == nil {
		log.Critical("nil sql connection")
		return fmt.Errorf("nil sql connection")
	}

	bkp, err := _sql3conns[1].Backup("main", _sql3conns[0], "main")
	if err != nil {
		return err
	}


	_, err = bkp.Step(-1)
	if err != nil {
		return err
	}

	bkp.Finish()
	log.Infof("synced <%s> to <%s>", src.Name, dbpath)

	return nil
}

func (dst *DB) SyncFromDisk(dbpath string) error {

	log.Debugf("Syncing <%s> to <%s>", dbpath, dst.Name)

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

	bk, err := _sql3conns[1].Backup("main", _sql3conns[0], "main")
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

// Copy from src DB to dst DB
// Source DB os overwritten
func (src *DB) CopyTo(dst *DB) {

	log.Debugf("Copying <%s> to <%s>", src.Name, dst.Name)

	srcDB, err := sqlx.Open(DriverBackupMode, src.Path)
	defer func() {
		srcDB.Close()
		_sql3conns = _sql3conns[:len(_sql3conns)-1]
	}()
	if err != nil {
		log.Error(err)
	}

	srcDB.Ping()

	dstDB, err := sqlx.Open(DriverBackupMode, dst.Path)
	defer func() {
		dstDB.Close()
		_sql3conns = _sql3conns[:len(_sql3conns)-1]
	}()
	if err != nil {
		log.Error(err)
	}
	dstDB.Ping()

	bk, err := _sql3conns[1].Backup("main", _sql3conns[0], "main")
	if err != nil {
		log.Error(err)
	}

	_, err = bk.Step(-1)
	if err != nil {
		log.Error(err)
	}

	bk.Finish()
}
