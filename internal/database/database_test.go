package database

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

const (
	TestDB = "./testdata/gosukidb_test.sqlite"
)

var (
	SchemaTestHooksEnabled bool
)

func TestNew(t *testing.T) {

	// Test buffer format
	t.Run("BufferPath", func(t *testing.T) {

		db := NewDB("buffer", "", DBTypeInMemoryDSN)

		if db.Path != "file:buffer?mode=memory&cache=shared&_journal=MEMORY" {
			t.Error("invalid buffer path")
		}

	})

	t.Run("MemPath", func(t *testing.T) {

		db := NewDB("cache", "", DBTypeCacheDSN)
		if db.Path != "file:cache?mode=memory&cache=shared&_journal=MEMORY" {
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

		if db.Path != "file:append_opts?mode=memory&cache=shared&_journal=MEMORY&foo=bar&mode=rw" {
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
			t.Error(err)
		}

		if err != ErrVfsLocked {
			t.Error(err)
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
			t.Error(err)
		}

		if err != ErrVfsLocked {
			t.Error(err)
		}

	})

}

func setupSyncTestDB(t *testing.T) (*DB, *DB) {
	if !SchemaTestHooksEnabled {
		RegisterSqliteHooks()
		SchemaTestHooksEnabled = true
	}
	srcDB, err := NewBuffer("test_src")
	if err != nil {
		t.Errorf("creating buffer: %s", err)
	}

	tmpDir := t.TempDir() // Create a temporary directory for the test database files
	dstPath := filepath.Join(tmpDir, "gosukidb_test.sqlite")
	dstDB := NewDB("test_sync_dst", dstPath, DBTypeFileDSN, DsnOptions{})
	initLocalDB(srcDB, dstPath)

	_, err = dstDB.Init()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		srcDB.Close()
		dstDB.Close()
		os.RemoveAll(tmpDir)
	})
	return srcDB, dstDB
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	err = out.Sync()
	if err != nil {
		return err
	}
	return nil
}

func TestSyncTo(t *testing.T) {
	srcDB, dstDB := setupSyncTestDB(t)

	bookmarks := []*RawBookmark{}
	modified := time.Now().Unix()
	for i := 1; i <= 10; i++ {
		url := fmt.Sprintf("http://example.com/bookmark%d", i)
		_, err := srcDB.Handle.Exec(
			`INSERT INTO gskbookmarks(
				url,
				metadata,
				tags,
				desc,
				modified,
				flags,
				module,
				xhsum
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			url,
			"title"+strconv.Itoa(i),
			"tag"+strconv.Itoa(i),
			"description"+strconv.Itoa(i),
			modified,
			0,
			"module"+strconv.Itoa(i),
			xhsum(
				url,
				"title"+strconv.Itoa(i),
				"tag"+strconv.Itoa(i),
				"description"+strconv.Itoa(i),
			),
		)
		if err != nil {
			t.Error(err)
		}
	}

	err := srcDB.Handle.Select(&bookmarks, `SELECT * FROM gskbookmarks`)
	require.NoError(t, err)

	// pretty.Print(bookmarks)
	srcDB.SyncTo(dstDB)

	// Check that dstDB contains the right data
	var count int
	err = dstDB.Handle.Get(&count, `SELECT COUNT(*) FROM gskbookmarks`)
	if err != nil {
		t.Error(err)
	}
	if count != len(bookmarks) {
		t.Errorf("Expected %d bookmarks in dstDB but got %d", len(bookmarks), count)
	}

	dstBookmarks := []*RawBookmark{}
	err = dstDB.Handle.Select(&dstBookmarks, `SELECT * FROM gskbookmarks`)
	if err != nil {
		t.Error(err)
	}

	// Compare the data in srcDB and dstDB for equality
	for i, bm := range bookmarks {
		if !reflect.DeepEqual(bm, dstBookmarks[i]) {
			t.Errorf(
				"Bookmark %d does not match: expected %+v but got %+v",
				i,
				bm,
				dstBookmarks[i],
			)
		}
	}
}

// func TestSyncFromGosukiDB(t *testing.T) {
// 	t.Skip("TODO: sync from gosuki db")
// }
//
// func TestSyncToGosukiDB(t *testing.T) {
// 	t.Skip("TODO: sync to gosuki db")
// }

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}
