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
	"path/filepath"

	"github.com/blob42/gosuki/internal/utils"
)

func Init() {
	var err error

	registerSqliteHooks()
	initCache()
	StartSyncScheduler()

	// Check and initialize local db as last step
	// browser bookmarks should already be in cache

	dbdir := utils.GetDefaultDBPath()
	dbpath := filepath.Join(dbdir, DBFileName)
	// Verifiy that local db directory path is writeable
	err = utils.CheckWriteable(dbdir)
	if err != nil {
		log.Critical(err)
	}

	// If local db exists load it to cacheDB
	var exists bool
	if exists, err = utils.CheckFileExists(dbpath); exists {
		if err != nil {
			log.Warning(err)
		}
		log.Infof("<%s> exists, preloading to cache", dbpath)
		er := Cache.DB.SyncFromDisk(dbpath)
		if er != nil {
			log.Critical(er)
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
		log.Critical(err)
	}

}
