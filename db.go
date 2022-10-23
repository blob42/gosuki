package main

import (
	"path/filepath"

	"git.sp4ke.xyz/sp4ke/gomark/database"
	"git.sp4ke.xyz/sp4ke/gomark/utils"
)

type DB = database.DB

func initDB() {
	var err error

	// Check and initialize local db as last step
	// browser bookmarks should already be in cache

	dbdir := utils.GetDefaultDBPath()
	dbpath := filepath.Join(dbdir, database.DBFileName)
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
		er := database.Cache.DB.SyncFromDisk(dbpath)
		if er != nil {
			log.Critical(er)
		}
	} else {
		if err != nil {
			log.Error(err)
		}

		// Else initialize it
		initLocalDB(database.Cache.DB, dbpath)
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
