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

// TODO: handle `modified` time
// sqlite database management
package database

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/blob42/gosuki/internal/logging"
	"github.com/blob42/gosuki/pkg/tree"

	"github.com/jmoiron/sqlx"
	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/sp4ke/hashmap"
)

var (
	//TODO!: document this
	_sql3conns           []*sqlite3.SQLiteConn // Only used for backup hook

	DefaultDBPath = "./"
)

type Index = *hashmap.RBTree
type Node = tree.Node

var log = logging.GetLogger("DB")

const (
	DBFileName = "gosuki.db"

	DBTypeFileDSN = "file:%s"

	DriverBackupMode = "sqlite_hook_backup"
	DriverDefault    = "sqlite3"
	GosukiMainTable  = "bookmarks"
)

type DBType int

const (
	DBTypeInMemory DBType = iota
	DBTypeRegularFile
)

// Differentiate between gosukidb.sqlite and other sqlite DBs
const (
	DBGosuki DBType = iota
	DBForeign
)

// Database schemas used for the creation of new databases
const (
	// metadata: name or title of resource
	// modified: time.Now().Unix()
	//
	// flags: designed to be extended in future using bitwise masks
	// Masks:
	//     0b00000001: set title immutable ((do not change title when updating the bookmarks from the web ))
	QCreateBookmarksTable = `
    CREATE TABLE if not exists bookmarks (
		id integer PRIMARY KEY,
		URL text NOT NULL UNIQUE,
		metadata text default '',
		tags text default '',
		desc text default '',
		modified integer default (strftime('%s')),
		flags integer default 0
	)
    `
)

type DsnOptions map[string]string

type DBError struct {
	// Database object where error occured
	DBName string

	// Error that occured
	Err error
}

func DBErr(dbName string, err error) DBError {
	return DBError{Err: err}
}

func (e DBError) Error() string {
	return fmt.Sprintf("<%s>: %s", e.DBName, e.Err)
}

var (
	ErrVfsLocked = errors.New("vfs locked")
)

type Opener interface {
	Open(driver string, dsn string) error
}

type SQLXOpener interface {
	Opener
	Get() *sqlx.DB
}

type SQLXDBOpener struct {
	handle *sqlx.DB
}

func (o *SQLXDBOpener) Open(driver string, dataSourceName string) error {
	var err error
	o.handle, err = sqlx.Open(driver, dataSourceName)
	if err != nil {
		return err
	}

	return nil
}

func (o *SQLXDBOpener) Get() *sqlx.DB {
	return o.handle
}

// DB encapsulates an sql.DB struct. All interactions with memory/buffer and
// disk databases are done through the DB object
type DB struct {
	Name       string
	Path       string
	Handle     *sqlx.DB
	EngineMode string
	AttachedTo []string
	Type       DBType

	filePath string

	SQLXOpener
	LockChecker
}

func (db *DB) open() error {
	var err error
	err = db.Open(db.EngineMode, db.Path)
	if err != nil {
		return err
	}

	db.Handle = db.Get()
	err = db.Handle.Ping()
	if err != nil {
		return err
	}

	log.Debugf("<%s> opened at <%s> with driver <%s>",
		db.Name,
		db.Path,
		db.EngineMode)

	return nil
}

func (db *DB) Locked() (bool, error) {
	return db.LockChecker.Locked()
}

// dbPath is empty string ("") when using in memory sqlite db
// Call to Init() required before using
func NewDB(name string, dbPath string, dbFormat string, opts ...DsnOptions) *DB {

	var path string
	var dbType DBType

	// Use name as path for  in memory database
	if dbPath == "" {
		path = fmt.Sprintf(dbFormat, name)
		dbType = DBTypeInMemory
	} else {
		path = fmt.Sprintf(dbFormat, dbPath)
		dbType = DBTypeRegularFile
	}

	// Handle DSN options
	if len(opts) > 0 {
		dsn := url.Values{}
		for _, o := range opts {
			for k, v := range o {
				dsn.Set(k, v)
			}
		}

		// Test if path has already query params
		pos := strings.IndexRune(path, '?')

		// Path already has query params
		if pos >= 1 {
			path = fmt.Sprintf("%s&%s", path, dsn.Encode()) //append
		} else {
			path = fmt.Sprintf("%s?%s", path, dsn.Encode())
		}

	}

	return &DB{
		Name:       name,
		Path:       path,
		Handle:     nil,
		EngineMode: DriverDefault,
		SQLXOpener: &SQLXDBOpener{},
		Type:       dbType,
		filePath:   dbPath,
		LockChecker: &VFSLockChecker{
			path: dbPath,
		},
	}

}

// We should export Open() in its own method and wrap
// with interface so we can mock it and test the lock status in Init()
// Initialize a sqlite database with Gosuki Schema if not already done
func (db *DB) Init() (*DB, error) {

	var err error

	if db.Handle != nil {
		log.Warningf("%s: already initialized", db)
		return db, nil
	}

	// Detect if database file is locked
	if db.Type == DBTypeRegularFile {

		if locked, err := db.Locked(); err != nil {
			return nil, DBError{DBName: db.Name, Err: err}
		} else if locked {
			return nil, ErrVfsLocked
		}
	}

	// Open database
	err = db.open()

	sqlErr, _ := err.(sqlite3.Error)

	// Secondary lock check provided by sqlx Ping() method
	if err != nil && sqlErr.Code == sqlite3.ErrBusy {
		return nil, ErrVfsLocked

	}

	// Return all other errors
	if err != nil {
		return nil, DBError{DBName: db.Name, Err: err}
	}


	return db, nil
}

func (db *DB) InitSchema() error {

	// Populate db schema
	tx, err := db.Handle.Begin()
	if err != nil {
		return DBError{DBName: db.Name, Err: err}
	}

	stmt, err := tx.Prepare(QCreateBookmarksTable)
	if err != nil {
		return DBError{DBName: db.Name, Err: err}
	}

	if _, err = stmt.Exec(); err != nil {
		return DBError{DBName: db.Name, Err: err}
	}

	if err = tx.Commit(); err != nil {
		return DBError{DBName: db.Name, Err: err}
	}

	log.Debugf("<%s> initialized", db.Name)

	return nil
}

func (db *DB) AttachTo(attached *DB) {

	stmtStr := fmt.Sprintf("ATTACH DATABASE '%s' AS '%s'",
		attached.Path,
		attached.Name)
	_, err := db.Handle.Exec(stmtStr)

	if err != nil {
		log.Error(err)
	}

	db.AttachedTo = append(db.AttachedTo, attached.Name)
}

func (db *DB) Close() error {
	log.Debugf("Closing DB <%s>", db.Name)

	if db.Handle == nil {
		log.Debugf("<%s> db handle is nil,  already closed ?", db.Name)
		return nil
	}

	err := db.Handle.Close()
	if err != nil {
		return err
	}
	return nil
}

func (db *DB) IsEmpty() (bool, error) {
	var count int

	row := db.Handle.QueryRow("select count(*) from bookmarks")

	err := row.Scan(&count)
	if err != nil {
		return false, err
	}

	if count > 0 {
		return false, nil
	}

	return true, nil
}

func (db *DB) CountRows(table string) int {
	var count int

	row := db.Handle.QueryRow("select count(*) from ?", table)
	err := row.Scan(&count)
	if err != nil {
		log.Error(err)
	}

	return count
}

// Struct represetning the schema of `bookmarks` db.
// The order in the struct respects the columns order
type SBookmark struct {
	id       int
	URL      string
	metadata string
	tags     string
	desc     string
	modified int64
	flags    int
}

// Scans a row into `SBookmark` schema
func ScanBookmarkRow(row *sql.Rows) (*SBookmark, error) {
	scan := new(SBookmark)
	err := row.Scan(
		&scan.id,
		&scan.URL,
		&scan.metadata,
		&scan.tags,
		&scan.desc,
		&scan.modified,
		&scan.flags,
	)

	if err != nil {
		return nil, err
	}

	return scan, nil
}

//TODO: doc
func flushSqliteCon(con *sqlx.DB) {
	con.Close()
	_sql3conns = _sql3conns[:len(_sql3conns)-1]
	// log.Debugf("Flushed sqlite conns -> %v", _sql3conns)
}

func registerSqliteHooks() {
	// sqlite backup hook
	// log.Debugf("backup_hook: registering driver %s", DriverBackupMode)
	// Register the hook
	sql.Register(DriverBackupMode,
		&sqlite3.SQLiteDriver{
			//TODO!: document
			//add extra connection that are used by the sql.Backup function
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				//log.Debugf("[ConnectHook] registering new connection")
				_sql3conns = append(_sql3conns, conn)
				// log.Debugf("[ConnectHook] registered new connection")
				// log.Debugf("%v", _sql3conns)
				return nil
			},
		})

}

//TODO!: manual initialization
// func init() {
// 	initCache()
// 	registerSqliteHooks()
// 	StartSyncScheduler()
// }
