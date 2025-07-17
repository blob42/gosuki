package database

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/blob42/gosuki/internal/utils"
)

// TestSchemaInitialization verifies that the database schema is properly initialized
// and that the schema version is correctly set.
func TestSchemaInitialization(t *testing.T) {
	if !SchemaTestHooksEnabled {
		RegisterSqliteHooks()
		SchemaTestHooksEnabled = true
	}
	dir := t.TempDir()
	dbPath := dir + "/test.db"

	// Initialize a new on disk database instance
	db, err := NewDB("test_db", "", DBTypeInMemoryDSN).Init()
	require.NoError(t, err, "failed to initialize memory database")
	db.backupToDisk(dbPath)

	// Refer to disk database
	db, err = NewDB("gosuki_db", dbPath, DBTypeFileDSN).Init()
	require.NoError(t, err, "failed to initialize disk database")

	err = db.InitSchema()
	require.NoError(t, err, "failed to initialize schema")

	// Verify that the schema version is set to the current version
	var version int
	err = db.Handle.QueryRow("SELECT version FROM schema_version").Scan(&version)
	require.NoError(t, err, "failed to query schema version")
	require.Equal(t, CurrentSchemaVersion, version, "schema version mismatch")

	// Verify that the required tables exist
	tables := []string{"gskbookmarks"}
	for _, table := range tables {
		var name string
		err = db.Handle.QueryRow(fmt.Sprintf(
			"SELECT NAME FROM sqlite_master WHERE type = 'table' AND name = '%s'",
			table,
		)).Scan(&name)
		require.NoError(t, err, "failed to check table `%s` existence", table)
		require.Equal(t, name, table, "table %s should exist", table)
	}

	// verify views exist
	views := []string{"bookmarks"}
	for _, view := range views {
		var name string
		err = db.Handle.QueryRow(fmt.Sprintf(
			"SELECT NAME FROM sqlite_master WHERE type = 'view' AND name = '%s'",
			view,
		)).Scan(&name)
		require.NoError(t, err, "failed to check table existence")
		require.Equal(t, name, view, "table %s should exist", view)
	}

	// Clean up
	db.Close()
	os.Remove(dbPath)
}

// TestSchemaUpgrade verifies that schema upgrades work correctly between versions.
func TestSchemaUpgrade(t *testing.T) {
	if !SchemaTestHooksEnabled {
		RegisterSqliteHooks()
		SchemaTestHooksEnabled = true
	}
	dir := t.TempDir()
	dbPath := dir + "/test.db"

	// Initialize a new on disk database instance
	db, err := NewDB("test_db", "", DBTypeInMemoryDSN).Init()
	require.NoError(t, err, "failed to initialize memory database")
	db.backupToDisk(dbPath)

	// Refer to ondisk db
	db, err = NewDB("test_db", dbPath, DBTypeFileDSN).Init()
	require.NoError(t, err, "failed to initialize disk database")

	// Manually create the old 'bookmarks' table (without versioning)
	_, err = db.Handle.Exec(`CREATE TABLE bookmarks (
		id integer PRIMARY KEY,
		URL text NOT NULL UNIQUE,
		metadata text default '',
		tags text default '',
		desc text default '',
		modified integer default (strftime('%s')),
		flags integer default 0,
		module text default ''
	)`)
	require.NoError(t, err, "failed to create old bookmarks table")

	// Clean up the database and re-initialize to simulate upgrade
	db.Close()

	//IMP: the db name must be "gosuki_db"
	db, err = NewDB("gosuki_db", dbPath, DBTypeFileDSN).Init()
	require.NoError(t, err, "failed to re-initialize database")

	err = checkDBVersion(db)
	require.NoError(t, err, "failed to check DB version before v0->v1 upgrade")

	// Verify that the schema version is set to 1
	var version int
	err = db.Handle.QueryRow("SELECT version FROM schema_version").Scan(&version)
	require.NoError(t, err, "failed to query schema version")
	require.Equal(t, CurrentSchemaVersion, version, "schema version upgrade failed")

	// Verify that the new tables exist after upgrade
	tables := []string{"gskbookmarks", "bookmarks"}
	for _, table := range tables {
		var count int
		err = db.Handle.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
		require.NoError(t, err, "failed to check table existence")
		require.GreaterOrEqual(t, count, 0, "table %s should exist after upgrade", table)
	}

	// Clean up
	db.Close()
	os.Remove(dbPath)
}

// TestSchemaVersionMismatch verifies the behavior when the stored schema version is invalid.
func TestSchemaVersionMismatch(t *testing.T) {
	if !SchemaTestHooksEnabled {
		RegisterSqliteHooks()
		SchemaTestHooksEnabled = true
	}

	dir := t.TempDir()
	dbPath := dir + "/test.db"

	// Initialize a new database instance
	db, err := NewDB("gosuki_db", "", DBTypeInMemoryDSN).Init()
	require.NoError(t, err, "failed to initialize database")

	err = db.InitSchema()
	require.NoError(t, err, "failed to initialize schema")

	// sync to disk
	err = db.backupToDisk(dbPath)
	require.NoError(t, err, "failed to sync memory db to disk")

	// refer to disk db
	db, err = NewDB("gosuki_db", dbPath, DBTypeFileDSN).Init()
	require.NoError(t, err, "failed to initialize disk db")

	// Simulate an invalid schema version
	_, err = db.Handle.Exec("UPDATE schema_version SET version = 999")
	require.NoError(t, err, "failed to set invalid schema version")
	utils.CopyFileToDst(dbPath, "/tmp/testgosuki.db")

	// Try to re-initialize the database and expect an error
	db.Close()
	db, err = NewDB("gosuki_db", dbPath, DBTypeFileDSN).Init()
	require.NoError(t, err, "failed to initialize disk db")
	err = checkDBVersion(db)
	require.Error(t, err, "expected error for invalid schema version")

	// Clean up
	db.Close()
	os.Remove(dbPath)
}

// TestSchemaVersionMissing verifies the behavior when the schema_version table is missing.
func TestSchemaVersionMissing(t *testing.T) {
	if !SchemaTestHooksEnabled {
		RegisterSqliteHooks()
		SchemaTestHooksEnabled = true
	}
	dir := t.TempDir()
	dbPath := dir + "/test.db"

	// Initialize a new database instance
	db, err := NewDB(dbPath, "", DBTypeInMemoryDSN).Init()
	require.NoError(t, err, "failed to initialize database")

	// Verify that the schema version is set to the current version
	var version int
	err = db.Handle.QueryRow("SELECT version FROM schema_version").Scan(&version)
	require.Error(t, err, "expected error for missing schema version table")

	err = checkDBVersion(db)
	require.NoError(t, err, "error checking db version")

	// Clean up
	os.Remove(dbPath)
}
