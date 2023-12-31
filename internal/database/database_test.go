package database

import (
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/mattn/go-sqlite3"
)

const (
	TestDB = "testdata/gosukidb_test.sqlite"
)

func TestNew(t *testing.T) {

	// Test buffer format
	t.Run("BufferPath", func(t *testing.T) {

		db := NewDB("buffer", "", DBTypeInMemoryDSN)

		if db.Path != "file:buffer?mode=memory&cache=shared" {
			t.Error("invalid buffer path")
		}

	})

	t.Run("MemPath", func(t *testing.T) {

		db := NewDB("cache", "", DBTypeCacheDSN)
		if db.Path != "file:cache?mode=memory&cache=shared" {
			t.Fail()
		}

	})

	t.Run("FilePath", func(t *testing.T) {

		db := NewDB("file_test", "/tmp/test/testdb.sqlite", DBTypeFileDSN)

		if db.Path != "file:/tmp/test/testdb.sqlite" {
			t.Fail()
		}

	})

	t.Run("FileCustomDsn", func(t *testing.T) {
		opts := DsnOptions{
			"foo":  "bar",
			"mode": "rw",
		}

		db := NewDB("file_dsn", "", DBTypeFileDSN, opts)

		if db.Path != "file:file_dsn?foo=bar&mode=rw" {
			t.Fail()
		}
	})

	t.Run("AppendOptions", func(t *testing.T) {
		opts := DsnOptions{
			"foo":  "bar",
			"mode": "rw",
		}

		db := NewDB("append_opts", "", DBTypeInMemoryDSN, opts)

		if db.Path != "file:append_opts?mode=memory&cache=shared&foo=bar&mode=rw" {
			t.Fail()
		}
	})
}

type AlwaysLockedChecker struct {
	locked bool
}

func (f *AlwaysLockedChecker) Locked() (bool, error) {
	return f.locked, nil
}

type LockedSQLXOpener struct {
	handle *sqlx.DB
	err    sqlite3.Error
}

func (o *LockedSQLXOpener) Open(driver string, dsn string) error {
	return o.err

}

func (o *LockedSQLXOpener) Get() *sqlx.DB {
	return nil
}

func TestInitLocked(t *testing.T) {
	lockedOpener := &LockedSQLXOpener{
		handle: nil,
		err:    sqlite3.Error{Code: sqlite3.ErrBusy},
	}

	lockCheckerTrue := &AlwaysLockedChecker{locked: true}
	lockCheckerFalse := &AlwaysLockedChecker{locked: false}

	t.Run("VFSLockChecker", func(t *testing.T) {

		testDB := &DB{
			Name:        "test",
			Path:        "file:test",
			EngineMode:  DriverDefault,
			LockChecker: lockCheckerTrue,
			SQLXOpener:  lockedOpener,
			Type:        DBTypeRegularFile,
		}

		_, err := testDB.Init()

		if err == nil {
			t.Fail()
		}

		if err != ErrVfsLocked {
			t.Fail()
		}

	})

	t.Run("SQLXLockChecker", func(t *testing.T) {

		testDB := &DB{
			Name:        "test",
			Path:        "file:test",
			EngineMode:  DriverDefault,
			LockChecker: lockCheckerFalse,
			SQLXOpener:  lockedOpener,
			Type:        DBTypeRegularFile,
		}

		_, err := testDB.Init()

		if err == nil {
			t.Fail()
		}

		if err != ErrVfsLocked {
			t.Fail()
		}

	})

}

func TestSyncFromGosukiDB(t *testing.T) {
	t.Skip("TODO: sync from gosuki db")
}

func TestSyncToGosukiDB(t *testing.T) {
	t.Skip("TODO: sync to gosuki db")
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}
