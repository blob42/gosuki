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

// sqlite database management
package database

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/OneOfOne/xxhash"
	"github.com/lithammer/fuzzysearch/fuzzy"

	"github.com/blob42/gosuki/internal/utils"
	"github.com/blob42/gosuki/pkg/config"
	"github.com/blob42/gosuki/pkg/logging"
	"github.com/blob42/gosuki/pkg/tree"

	"github.com/blob42/hashmap"
	"github.com/jmoiron/sqlx"
	sqlite3 "github.com/mattn/go-sqlite3"
)

var (
	// DOC:

	// List of sql connections, used to do a sql backup
	_sql3BackupConns []*sqlite3.SQLiteConn

	// Handle to on-disk gosuki database
	DiskDB *DB

	Config *dbConfig
)

// Index is a RedBlack Tree Hashmap that holds in memory the last state of the
// bookmark tree.It is used as fast db queries. Each URL holds a pointer to a
// node in [nodeTree]
type Index = *hashmap.RBTree

// Node is a URLTree
type Node = tree.Node

var (
	log = logging.GetLogger("db")

	// Default sqlite3 driver
	DriverDefault = "sqlite3_gosuki"
)

const (
	DBFileName = "gosuki.db"

	DBTypeFileDSN = "file:%s"

	// Opening DBs with this driver allows to track connections
	// This is used to perform sqlite backup
	DriverBackupMode = "sqlite_hook_backup"

	GosukiMainTable = "bookmarks"
)

type DBType int

const (
	DBTypeInMemory DBType = iota
	DBTypeRegularFile
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
// disk databases are done through the DB instance.
type DB struct {
	Name       string
	Path       string
	Handle     *sqlx.DB
	EngineMode string
	AttachedTo []string
	Type       DBType
	mu         *sync.RWMutex

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

	// Use name as path for in memory database
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
		mu:         &sync.RWMutex{},
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
		log.Warn("db already initialized", "db", db.Name)
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

func (db *DB) Attach(attached *DB) error {

	stmtStr := fmt.Sprintf("ATTACH DATABASE '%s' AS '%s'",
		attached.Path,
		attached.Name)
	_, err := db.Handle.Exec(stmtStr)

	if err != nil {
		return err
	}

	db.AttachedTo = append(db.AttachedTo, attached.Name)
	return nil
}

func (db *DB) Close() error {
	log.Debugf("closing DB <%s>", db.Name)

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

func GetDBPath() string {
	var err error

	// Check and initialize local db as last step
	// browser bookmarks should already be in cache
	dbdir := GetDBDir()
	err = utils.MkDir(dbdir)
	if err != nil {
		log.Error(err)
	}

	dbpath := filepath.Join(dbdir, DBFileName)
	// Verifiy that local db directory path is writeable
	err = utils.MkDir(dbdir)
	if err != nil {
		log.Error(err)
	}

	return dbpath
}

// flushSqliteCon closes a SQLite database connection and removes it from the internal list of connections.
func flushSqliteCon(con *sqlx.DB) {
	con.Close()
	_sql3BackupConns = _sql3BackupConns[:len(_sql3BackupConns)-1]
	// log.Debugf("Flushed sqlite conns -> %v", _sql3conns)
}

// Testing custom func
func SQLFuncFoo(in string) string {
	return strings.ToUpper(in + "foo")
}

func SQLFuzzy(test, in string) bool {
	return fuzzy.MatchFold(test, in)
}

func SQLxxHash(in string) string {
	return fmt.Sprintf("%d", xxhash.ChecksumString64(in))
}

// Calculates xxhash sum for a bookmark
func xhsum(url, metadata, tags, desc string) string {
	input := fmt.Sprintf(
		"%s+%s+%s+%s",
		url,
		metadata,
		tags,
		desc,
	)
	return SQLxxHash(input)
}

// ticks the local node's lamport clock and returns current clock number
func sqlTickClock(previous uint64) uint64 {
	return Clock.Tick(previous)
}

// RegisterSqliteHooks registers a SQLite backup hook with additional connection tracking.
func RegisterSqliteHooks() {

	sql.Register(DriverDefault,
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				// if err := conn.RegisterFunc("sqlfoo", SQLFuncFoo, true); err != nil {
				// 	return err
				// }

				if err := conn.RegisterFunc("fuzzy", SQLFuzzy, true); err != nil {
					return err
				}

				// register function that will update internal clock
				if err := conn.RegisterFunc("tick_clock", sqlTickClock, true); err != nil {
					return err
				}

				return conn.RegisterFunc("xhash", SQLxxHash, true)

			},
		})

	// sqlite backup hook
	// log.Debugf("backup_hook: registering driver %s", DriverBackupMode)
	sql.Register(DriverBackupMode,
		&sqlite3.SQLiteDriver{
			// ConnectHook is a function that is called when a new connection to the SQLite database is established.
			// See: https://github.com/mattn/go-sqlite3/blob/82bc911e85b3def2940e41767480745bcbb6ef45/_example/hook/hook.go#L59
			// See: sync.go: SyncToDisk
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				//log.Debugf("[ConnectHook] registering new connection")
				_sql3BackupConns = append(_sql3BackupConns, conn)
				// log.Debugf("[ConnectHook] registered new connection")
				// log.Debugf("%v", _sql3conns)
				return nil
			},
		})
}

type dbConfig struct {
	SyncInterval time.Duration `toml:"sync-interval" mapstructure:"sync-interval"`
	Path         string        `toml:"path" mapstructure:"path"`
}

func init() {
	var dataDir string
	var err error

	if dataDir, err = utils.GetDataDir(); err != nil {
		log.Fatal(err)
	}

	dbPath := filepath.Join(dataDir, "gosuki/gosuki.db")

	Config = &dbConfig{
		SyncInterval: time.Second * 4,
		Path:         dbPath,
	}

	config.RegisterConfigurator("database", config.AsConfigurator(Config))

}
