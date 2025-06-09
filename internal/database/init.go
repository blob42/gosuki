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

import (
	"github.com/blob42/gosuki/internal/utils"
)

// Initialize the connection to ondisk gosuki db
func InitDiskConn(dbPath string) error {
	log.Debug("initializing cacheDB")
	var err error

	// Initialize connection to gosuki file database

	dsnOpts := DsnOptions{
		"_journal_mode": "WAL",
		// see https://github.com/mattn/go-sqlite3/issues/249
		"_mutex": "no",
		"cache":  "shared",
	}
	DiskDB, err = NewDB("gosuki_db", dbPath, DBTypeFileDSN, dsnOpts).Init()
	DiskDB.Handle.Exec("PRAGMA wal_checkpoint(TRUNCATE)")

	return err
}

func Init() {

	RegisterSqliteHooks()
	initCache()
	StartSyncScheduler()

	dbpath := GetDBPath()
	// If local db exists load it to cacheDB
	if exists, err := utils.CheckFileExists(dbpath); exists {
		log.Infof("<%s> exists, preloading to cache", dbpath)
		err = InitDiskConn(dbpath)
		if err != nil {
			log.Error(err)
		}
		err = Cache.SyncFromDisk(dbpath)
		if err != nil {
			log.Error(err)
		}
	} else {
		if err != nil {
			log.Error(err)
		}

		// Else initialize it
		initLocalDB(Cache.DB, dbpath)
	}

}

// Initialize the local database file
func initLocalDB(db *DB, dbpath string) {

	log.Infof("Initializing local db at '%s'", dbpath)
	err := db.SyncToDisk(dbpath)
	if err != nil {
		log.Fatal(err)
	}

	err = InitDiskConn(dbpath)
	if err != nil {
		log.Fatal(err)
	}
}
