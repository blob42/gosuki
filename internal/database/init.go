//
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

package database

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/blob42/gosuki/internal/utils"
	"github.com/blob42/gosuki/pkg/config"
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
	if err != nil {
		return err
	}

	_, err = DiskDB.Handle.Exec("PRAGMA wal_checkpoint(TRUNCATE)")

	return err
}

func Init(ctx context.Context, cmd *cli.Command) {
	var err error

	RegisterSqliteHooks()
	initCache(ctx)
	startSchedulers()

	config.DBPath, err = utils.ExpandOnly(config.DBPath)
	if err != nil {
		log.Fatal(err)
	}
	dbpath := config.DBPath
	if dbpath == "" {
		log.Fatal("undefined database path")
	}

	// If local db exists load it to cacheDB
	if exists, err := utils.CheckFileExists(dbpath); exists {
		if err != nil {
			log.Fatal(err)
		}

		log.Infof("preloading <%s> to cache", utils.Shorten(dbpath))
		err = InitDiskConn(dbpath)
		if err != nil {
			log.Fatal(err)
		}

		err = checkDBVersion(DiskDB)
		if err != nil {
			log.Fatal(err)
		}

		// first sync to the l1 cache from disk
		err = Cache.SyncFromDisk(dbpath)
		if err != nil {
			log.Fatal(err)
		}

		// Also sync l2 cache from disk
		err = L2Cache.SyncFromDisk(dbpath)
		if err != nil {
			log.Fatal(err)
		}

	} else {
		// new pristine db
		if err != nil {
			log.Fatal(err)
		}

		// Else initialize it
		initLocalDB(Cache.DB, dbpath)
	}

	// init local lamport clock
	Clock, err = L2Cache.GetDBClock(ctx)
	if err != nil {
		log.Fatalf("getting local db clock: %s", err)
	}
}

// Initialize the local database file
func initLocalDB(db *DB, dbpath string) {

	log.Infof("Initializing local db at '%s'", dbpath)
	err := db.BackupToDisk(dbpath)
	if err != nil {
		log.Fatal(err)
	}

	err = InitDiskConn(dbpath)
	if err != nil {
		log.Fatal(err)
	}
}
