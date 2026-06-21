// Package test provides a shared test harness for gosuki CLI and database tests.
//
// All e2e and integration tests should use this harness to create isolated,
// reproducible test environments with known seed data.
package test

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	db "github.com/blob42/gosuki/internal/database"
	"github.com/blob42/gosuki/test/fixtures"
)

var hooksOnce sync.Once

// Harness holds test resources that need cleanup.
type Harness struct {
	DBPath string // Path to temp SQLite database file
	Dir    string // Temp directory (auto-cleaned by testing.T)
	DB     *db.DB // Initialized database handle
}

// NewHarness creates a fresh test database with schema initialized.
//
// It uses t.TempDir() for automatic cleanup on test completion and the
// in-memory-to-disk backup pattern from the existing test suite.
//
// The returned harness has a fully initialized database (schema v4) ready
// for seeding. Call h.SeedBookmarks() to populate it with test data.
func NewHarness(t testing.TB) *Harness {
	t.Helper()

	// Register sqlite hooks (fuzzy search, xxhash, etc.) required by schema.
	// Use sync.Once because sql.Register panics on duplicate registration.
	hooksOnce.Do(db.RegisterSqliteHooks)

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// Create in-memory DB, initialize schema, then backup to disk.
	// This mirrors the pattern used in schema_test.go and sync_test.go.
	memDB, err := db.NewDB("mem", "", db.DBTypeInMemoryDSN).Init()
	if err != nil {
		t.Fatalf("failed to create memory DB: %v", err)
	}

	err = memDB.InitSchema(context.Background())
	if err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	memDB.BackupToDisk(dbPath)
	_ = memDB.Close()

	// Open the disk-backed DB for test use
	diskDB, err := db.NewDB("test_db", dbPath, db.DBTypeFileDSN).Init()
	if err != nil {
		t.Fatalf("failed to init disk DB: %v", err)
	}

	return &Harness{
		DBPath: dbPath,
		Dir:    dir,
		DB:     diskDB,
	}
}

// Cleanup closes the database handle. The temp directory is auto-cleaned
// by testing.T when the test completes.
func (h *Harness) Cleanup() {
	if h.DB != nil && h.DB.Handle != nil {
		h.DB.Handle.Close()
	}
}

// SeedBookmarks inserts bookmarks into the test database.
// Returns the number of bookmarks inserted.
// Panics on SQL errors (tests should not recover from seed failures).
func (h *Harness) SeedBookmarks(bookmarks []fixtures.SeedBookmark) int {
	for _, bm := range bookmarks {
		stmt := fixtures.BookmarkToInsertSQL(bm)
		if _, err := h.DB.Handle.Exec(stmt); err != nil {
			panic(fmt.Sprintf("seed insert failed: %v (url: %s)", err, bm.URL))
		}
	}
	return len(bookmarks)
}

// SeedAndCount is a convenience method that seeds bookmarks and returns
// the total row count in gskbookmarks. Useful for verifying insertion.
func (h *Harness) SeedAndCount(bookmarks []fixtures.SeedBookmark) int {
	h.SeedBookmarks(bookmarks)
	var count int
	err := h.DB.Handle.Get(&count, "SELECT COUNT(*) FROM gskbookmarks")
	if err != nil {
		panic(fmt.Sprintf("seed count failed: %v", err))
	}
	return count
}

// RunWithDB creates a harness, runs the test function, and cleans up.
// Use this for one-off tests that don't need the harness elsewhere.
//
// Example:
//
//	test.RunWithDB(t, func(h *test.Harness) {
//	    h.SeedBookmarks(test.DefaultSeedSet())
//	    // ... test logic ...
//	})
func RunWithDB(t testing.TB, fn func(*Harness)) {
	t.Helper()
	h := NewHarness(t)
	defer h.Cleanup()
	fn(h)
}

// NewHarnessWithSeed is a convenience that creates a harness and seeds it
// in one call.
func NewHarnessWithSeed(t testing.TB, bookmarks []fixtures.SeedBookmark) *Harness {
	t.Helper()
	h := NewHarness(t)
	h.SeedBookmarks(bookmarks)
	return h
}
