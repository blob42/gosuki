//go:build integration

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	db "github.com/blob42/gosuki/internal/database"
	"github.com/blob42/gosuki/internal/utils"
	"github.com/blob42/gosuki/test"
	"github.com/blob42/gosuki/test/fixtures"
)

// TestPathExpansionTilde verifies that ~ in the database path is expanded
// before opening the database.
//
// Regression test for https://github.com/blob42/gosuki/issues/25
// Without the fix, suki tries to open a literal file starting with "~"
// which either fails with "no such file" or creates an empty DB causing
// a nil pointer panic in ListBookmarks.
func TestPathExpansionTilde(t *testing.T) {
	h := test.NewHarnessWithSeed(t, fixtures.DefaultSeedSet())
	defer h.Cleanup()

	// Copy the test DB to a location under $HOME
	testSubDir := ".gosuki-test-suki-tilde"
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	absDir := filepath.Join(homeDir, testSubDir)
	require.NoError(t, os.MkdirAll(absDir, 0755))
	t.Cleanup(func() { os.RemoveAll(absDir) })

	dbPath := filepath.Join(absDir, "test.db")
	require.NoError(t, h.DB.BackupToDisk(dbPath))

	// Simulate the config value: path with ~
	rawPath := filepath.Join("~", testSubDir, "test.db")

	// Expand the path (this is what the fix does in suki Before hook)
	expanded, err := utils.ExpandOnly(rawPath)
	require.NoError(t, err)
	require.Equal(t, dbPath, expanded, "expanded path should match expected absolute path")

	// Open the database with the expanded path — this is the critical step
	// that failed before the fix (literal "~" passed to SQLite)
	conn, err := db.NewDB("test", expanded, db.DBTypeFileDSN).Init()
	require.NoError(t, err, "should open database at expanded path")
	defer conn.Handle.Close()

	// Verify we can actually query the database
	var count int
	err = conn.Handle.Get(&count, "SELECT COUNT(*) FROM gskbookmarks")
	require.NoError(t, err)
	require.Equal(t, 5, count, "should have seeded bookmarks")
}

// TestPathExpansionDollarHome verifies that $HOME in the database path is
// expanded before opening the database.
//
// Regression test for https://github.com/blob42/gosuki/issues/25
func TestPathExpansionDollarHome(t *testing.T) {
	h := test.NewHarnessWithSeed(t, fixtures.DefaultSeedSet())
	defer h.Cleanup()

	// Copy the test DB to a location under $HOME
	testSubDir := ".gosuki-test-suki-home"
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	absDir := filepath.Join(homeDir, testSubDir)
	require.NoError(t, os.MkdirAll(absDir, 0755))
	t.Cleanup(func() { os.RemoveAll(absDir) })

	dbPath := filepath.Join(absDir, "test.db")
	require.NoError(t, h.DB.BackupToDisk(dbPath))

	// Simulate the config value: path with $HOME
	rawPath := "$HOME/" + testSubDir + "/test.db"

	// Expand the path (this is what the fix does in suki Before hook)
	expanded, err := utils.ExpandOnly(rawPath)
	require.NoError(t, err)
	require.Equal(t, dbPath, expanded, "expanded path should match expected absolute path")

	// Open the database with the expanded path
	conn, err := db.NewDB("test", expanded, db.DBTypeFileDSN).Init()
	require.NoError(t, err, "should open database at expanded path")
	defer conn.Handle.Close()

	// Verify we can actually query the database
	var count int
	err = conn.Handle.Get(&count, "SELECT COUNT(*) FROM gskbookmarks")
	require.NoError(t, err)
	require.Equal(t, 5, count, "should have seeded bookmarks")
}

// TestListBookmarksAfterExpand verifies the full suki code path:
// expand path -> InitDiskConn -> ListBookmarks (the exact chain that
// panicked before the fix).
func TestListBookmarksAfterExpand(t *testing.T) {
	h := test.NewHarnessWithSeed(t, fixtures.DefaultSeedSet())
	defer h.Cleanup()

	testSubDir := ".gosuki-test-suki-list"
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	absDir := filepath.Join(homeDir, testSubDir)
	require.NoError(t, os.MkdirAll(absDir, 0755))
	t.Cleanup(func() { os.RemoveAll(absDir) })

	dbPath := filepath.Join(absDir, "test.db")
	require.NoError(t, h.DB.BackupToDisk(dbPath))

	// Simulate the full suki Before hook path:
	// 1. Raw path from config with ~
	rawPath := filepath.Join("~", testSubDir, "test.db")

	// 2. Expand (the fix)
	expanded, err := utils.ExpandOnly(rawPath)
	require.NoError(t, err)

	// 3. InitDiskConn (this is what suki calls)
	err = db.InitDiskConn(expanded)
	require.NoError(t, err)

	// 4. ListBookmarks (this panicked with nil DiskDB.Handle before the fix)
	result, err := db.ListBookmarks(context.Background(), &db.PaginationParams{
		Page: 1,
		Size: -1,
	})
	require.NoError(t, err, "ListBookmarks should not panic")
	require.Equal(t, 5, len(result.Bookmarks), "should list all seeded bookmarks")
}
