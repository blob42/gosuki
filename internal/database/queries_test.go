package database

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/blob42/gosuki/test/fixtures"
)

// newTestDB creates a disk-backed test DB with schema initialized, swaps DiskDB,
// and returns a cleanup function. Call defer db, cleanup := newTestDB(t).
func newTestDB(t *testing.T) (*DB, func()) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	memDB, err := NewDB("mem", "", DBTypeInMemoryDSN).Init()
	require.NoError(t, err)
	err = memDB.InitSchema(context.Background())
	require.NoError(t, err)
	memDB.BackupToDisk(dbPath)
	memDB.Close()

	diskDB, err := NewDB("test_db", dbPath, DBTypeFileDSN).Init()
	require.NoError(t, err)

	orig := DiskDB
	DiskDB = diskDB
	return diskDB, func() {
		DiskDB = orig
		diskDB.Close()
	}
}

// seedDB inserts seed bookmarks into the current DiskDB and returns a no-op cleanup.
func seedDB(t *testing.T, db *DB, bookmarks []fixtures.SeedBookmark) func() {
	t.Helper()
	for _, bm := range bookmarks {
		stmt := fixtures.BookmarkToInsertSQL(bm)
		_, err := db.Handle.Exec(stmt)
		require.NoError(t, err, "seed insert failed for %s", bm.URL)
	}
	return func() {}
}

// --- buildOrderBy tests (pure function, no DB needed) ---

func TestBuildOrderBy_NoSort(t *testing.T) {
	require.Equal(t, "", buildOrderBy(nil), "nil pagination should return empty string")
	require.Equal(t, "", buildOrderBy(&PaginationParams{}), "empty SortBy should return empty string")
}

func TestBuildOrderBy_InvalidField(t *testing.T) {
	p := &PaginationParams{SortBy: "invalid_field"}
	require.Equal(t, "", buildOrderBy(p), "invalid field should return empty string")

	p = &PaginationParams{SortBy: "DROP TABLE"}
	require.Equal(t, "", buildOrderBy(p), "SQL injection attempt should return empty string")
}

func TestBuildOrderBy_ValidFields(t *testing.T) {
	tests := []struct {
		name    string
		sortBy  string
		sortAsc bool
		want    string
	}{
		{"modified desc", "modified", false, " ORDER BY modified DESC"},
		{"modified asc", "modified", true, " ORDER BY modified ASC"},
		{"title desc", "title", false, " ORDER BY metadata DESC"},
		{"title asc", "title", true, " ORDER BY metadata ASC"},
		{"url desc", "url", false, " ORDER BY url DESC"},
		{"url asc", "url", true, " ORDER BY url ASC"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &PaginationParams{SortBy: tc.sortBy, SortAsc: tc.sortAsc}
			require.Equal(t, tc.want, buildOrderBy(p))
		})
	}
}

// --- ListBookmarks sort tests (integration, needs DiskDB swap) ---

func TestListBookmarks_SortModifiedDesc(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.DefaultSeedSet())()

	result, err := ListBookmarks(context.Background(), &PaginationParams{
		Page: 1, Size: -1, SortBy: "modified", SortAsc: false,
	})
	require.NoError(t, err)
	require.Equal(t, 5, len(result.Bookmarks))

	// Desc: Epsilon(5000), Delta(4000), Gamma(3000), Beta(2000), Alpha(1000)
	expected := []string{"Epsilon", "Delta", "Gamma", "Beta", "Alpha"}
	for i, bm := range result.Bookmarks {
		require.Equal(t, expected[i], bm.Title, "index %d", i)
	}
}

func TestListBookmarks_SortModifiedAsc(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.DefaultSeedSet())()

	result, err := ListBookmarks(context.Background(), &PaginationParams{
		Page: 1, Size: -1, SortBy: "modified", SortAsc: true,
	})
	require.NoError(t, err)
	require.Equal(t, 5, len(result.Bookmarks))

	// Asc: Alpha(1000), Beta(2000), Gamma(3000), Delta(4000), Epsilon(5000)
	expected := []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon"}
	for i, bm := range result.Bookmarks {
		require.Equal(t, expected[i], bm.Title, "index %d", i)
	}
}

func TestListBookmarks_SortTitleAsc(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.DefaultSeedSet())()

	result, err := ListBookmarks(context.Background(), &PaginationParams{
		Page: 1, Size: -1, SortBy: "title", SortAsc: true,
	})
	require.NoError(t, err)
	require.Equal(t, 5, len(result.Bookmarks))

	// Alpha, Beta, Delta, Epsilon, Gamma
	expected := []string{"Alpha", "Beta", "Delta", "Epsilon", "Gamma"}
	for i, bm := range result.Bookmarks {
		require.Equal(t, expected[i], bm.Title, "index %d", i)
	}
}

func TestListBookmarks_SortURLAsc(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.DefaultSeedSet())()

	result, err := ListBookmarks(context.Background(), &PaginationParams{
		Page: 1, Size: -1, SortBy: "url", SortAsc: true,
	})
	require.NoError(t, err)
	require.Equal(t, 5, len(result.Bookmarks))

	// alpha.com, beta.com, delta.com, epsilon.com, gamma.com
	expected := []string{"Alpha", "Beta", "Delta", "Epsilon", "Gamma"}
	for i, bm := range result.Bookmarks {
		require.Equal(t, expected[i], bm.Title, "index %d", i)
	}
}

func TestListBookmarks_SortURLDesc(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.DefaultSeedSet())()

	result, err := ListBookmarks(context.Background(), &PaginationParams{
		Page: 1, Size: -1, SortBy: "url", SortAsc: false,
	})
	require.NoError(t, err)
	require.Equal(t, 5, len(result.Bookmarks))

	// gamma.com, epsilon.com, delta.com, beta.com, alpha.com
	expected := []string{"Gamma", "Epsilon", "Delta", "Beta", "Alpha"}
	for i, bm := range result.Bookmarks {
		require.Equal(t, expected[i], bm.Title, "index %d", i)
	}
}

func TestListBookmarks_NoSort(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.DefaultSeedSet())()

	result, err := ListBookmarks(context.Background(), &PaginationParams{
		Page: 1, Size: -1,
	})
	require.NoError(t, err)
	require.Equal(t, 5, len(result.Bookmarks))
}

func TestListBookmarks_SortInvalidField(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.DefaultSeedSet())()

	// Invalid field silently ignored (no sort), still returns all results
	result, err := ListBookmarks(context.Background(), &PaginationParams{
		Page: 1, Size: -1, SortBy: "not_a_column", SortAsc: false,
	})
	require.NoError(t, err)
	require.Equal(t, 5, len(result.Bookmarks))
}
