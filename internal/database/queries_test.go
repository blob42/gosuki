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

// --- DefaultPagination ---

func TestDefaultPagination(t *testing.T) {
	p := DefaultPagination()
	require.Equal(t, 1, p.Page)
	require.Equal(t, 50, p.Size)
	require.Equal(t, "", p.SortBy)
	require.False(t, p.SortAsc)
}

// --- buildWhereClause (pure function) ---

func TestBuildWhereClause_NoTagNoFuzzy(t *testing.T) {
	w := buildWhereClause("", false)
	require.Contains(t, w, "URL like")
	require.Contains(t, w, "metadata like")
	require.Contains(t, w, "LOWER(tags) like")
}

func TestBuildWhereClause_TagNoFuzzy(t *testing.T) {
	w := buildWhereClause("linux", false)
	require.Contains(t, w, "URL LIKE")
	require.Contains(t, w, "metadata LIKE")
	require.Contains(t, w, "LOWER(tags) LIKE")
}

func TestBuildWhereClause_NoTagFuzzy(t *testing.T) {
	w := buildWhereClause("", true)
	require.Contains(t, w, "fuzzy(")
}

func TestBuildWhereClause_TagFuzzy(t *testing.T) {
	w := buildWhereClause("linux", true)
	require.Contains(t, w, "fuzzy(")
	require.Contains(t, w, "LOWER(tags) LIKE")
}

// --- buildCountQuery (pure function) ---

func TestBuildCountQuery_NoTag(t *testing.T) {
	q := buildCountQuery("", false, "test", "test")
	require.Contains(t, q, "SELECT COUNT(*)")
	require.Contains(t, q, "gskbookmarks")
	require.Contains(t, q, "LIMIT 1")
}

func TestBuildCountQuery_Tag(t *testing.T) {
	q := buildCountQuery("linux", false, "test", "linux")
	require.Contains(t, q, "SELECT COUNT(*)")
	require.Contains(t, q, "LOWER(tags) LIKE")
}

func TestBuildCountQuery_Fuzzy(t *testing.T) {
	q := buildCountQuery("", true, "test", "test")
	require.Contains(t, q, "fuzzy(")
}

// --- buildWhereClauseForManyTags (pure function) ---

func TestBuildWhereClauseForManyTags_NoQuery(t *testing.T) {
	w := buildWhereClauseForManyTags("", []string{"linux", "os"}, TagAnd, false)
	require.Contains(t, w, "LOWER(tags) like '%linux%'")
	require.Contains(t, w, "LOWER(tags) like '%os%'")
	require.Contains(t, w, "AND")
}

func TestBuildWhereClauseForManyTags_WithQuery(t *testing.T) {
	w := buildWhereClauseForManyTags("lang", []string{"programming"}, TagAnd, false)
	require.Contains(t, w, "URL like '%lang%'")
	require.Contains(t, w, "LOWER(tags) like '%programming%'")
}

func TestBuildWhereClauseForManyTags_OrCondition(t *testing.T) {
	w := buildWhereClauseForManyTags("", []string{"linux", "os"}, TagOr, false)
	require.Contains(t, w, "OR")
	require.Contains(t, w, "LOWER(tags) like '%linux%'")
	require.Contains(t, w, "LOWER(tags) like '%os%'")
}

func TestBuildWhereClauseForManyTags_Fuzzy(t *testing.T) {
	w := buildWhereClauseForManyTags("", []string{"go"}, TagAnd, true)
	require.Contains(t, w, "fuzzy('go', tags)")
}

// --- buildSelectQuery (pure function) ---

func TestBuildSelectQuery_NoTag(t *testing.T) {
	q := buildSelectQuery("test", false, "", DefaultPagination())
	require.Contains(t, q, "SELECT URL, metadata, tags, module")
	require.Contains(t, q, "gskbookmarks")
	require.Contains(t, q, "URL like '%test%'")
	require.Contains(t, q, "LIMIT")
}

func TestBuildSelectQuery_WithTag(t *testing.T) {
	q := buildSelectQuery("test", false, "linux", DefaultPagination())
	require.Contains(t, q, "URL LIKE '%test%'")
	require.Contains(t, q, "LOWER(tags) LIKE '%linux%'")
}

func TestBuildSelectQuery_Fuzzy(t *testing.T) {
	q := buildSelectQuery("test", true, "", DefaultPagination())
	require.Contains(t, q, "fuzzy('test', URL)")
}

// --- QueryBookmarks (integration) ---

func TestQueryBookmarks_ByTitle(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.DefaultSeedSet())()

	result, err := QueryBookmarks(context.Background(), "Alpha", false, DefaultPagination())
	require.NoError(t, err)
	require.Equal(t, 1, len(result.Bookmarks))
	require.Equal(t, "Alpha", result.Bookmarks[0].Title)
	require.Equal(t, uint(1), result.Total)
}

func TestQueryBookmarks_ByURL(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.DefaultSeedSet())()

	result, err := QueryBookmarks(context.Background(), "beta.com", false, DefaultPagination())
	require.NoError(t, err)
	require.Equal(t, 1, len(result.Bookmarks))
	require.Equal(t, "Beta", result.Bookmarks[0].Title)
}

func TestQueryBookmarks_EmptyQuery(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.DefaultSeedSet())()

	_, err := QueryBookmarks(context.Background(), "", false, DefaultPagination())
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty")
}

func TestQueryBookmarks_Fuzzy(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.DefaultSeedSet())()

	result, err := QueryBookmarks(context.Background(), "alph", true, DefaultPagination())
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Bookmarks), 1)
}

// --- QueryBookmarksByTag (integration) ---
// Only error-path tests are included until the bug is fixed.

func TestQueryBookmarksByTag_EmptyQuery(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.DefaultSeedSet())()

	_, err := QueryBookmarksByTag(context.Background(), "", "a", false, DefaultPagination())
	require.Error(t, err)
}

func TestQueryBookmarksByTag_EmptyTag(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.DefaultSeedSet())()

	_, err := QueryBookmarksByTag(context.Background(), "test", "", false, DefaultPagination())
	require.Error(t, err)
}

func TestQueryBookmarksByTag_NilPagination(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.DefaultSeedSet())()

	_, err := QueryBookmarksByTag(context.Background(), "test", "a", false, nil)
	require.Error(t, err)
}

// --- QueryBookmarksByTags (integration) ---
// clause contains % wildcards that get consumed by Sprintf. Functional tests
// are skipped until the bug is fixed in queries.go.

func TestQueryBookmarksByTags_EmptyTags(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.DefaultSeedSet())()

	_, err := QueryBookmarksByTags(context.Background(), "test", []string{}, TagAnd, false, DefaultPagination())
	require.Error(t, err)
}

func TestQueryBookmarksByTags_NilPagination(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.DefaultSeedSet())()

	_, err := QueryBookmarksByTags(context.Background(), "test", []string{"a"}, TagAnd, false, nil)
	require.Error(t, err)
}

// --- BookmarksByTag (integration) ---
// contains LIKE wildcards (%%) that get consumed by fmt.Sprintf.
// Only error-path tests are included until the bug is fixed.

func TestBookmarksByTag_EmptyTag(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.DefaultSeedSet())()

	_, err := BookmarksByTag(context.Background(), "", DefaultPagination())
	require.Error(t, err)
}

// --- BookmarksByTags (integration) ---
// and crashes on nil pagination. Only error-path tests are included.

func TestBookmarksByTags_EmptyTags(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.DefaultSeedSet())()

	_, err := BookmarksByTags(context.Background(), []string{}, TagAnd, DefaultPagination())
	require.Error(t, err)
}

func TestQueryBookmarksByTag_Match(t *testing.T) {
	//
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.TagVarietySet())()

	result, err := QueryBookmarksByTag(context.Background(), "lang", "programming", false, DefaultPagination())
	require.NoError(t, err)
	require.Equal(t, 2, len(result.Bookmarks))
	require.Equal(t, uint(2), result.Total)
}
func TestQueryBookmarksByTag_NoMatch(t *testing.T) {
	//
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.DefaultSeedSet())()

	result, err := QueryBookmarksByTag(context.Background(), "nonexistent", "a", false, DefaultPagination())
	require.NoError(t, err)
	require.Equal(t, 0, len(result.Bookmarks))
	require.Equal(t, uint(0), result.Total)
}
func TestQueryBookmarksByTags_And(t *testing.T) {
	//
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.TagVarietySet())()

	// Linux (linux, os) and GNU (linux, gnu, os) both have "linux" AND "os"
	result, err := QueryBookmarksByTags(context.Background(), "", []string{"linux", "os"}, TagAnd, false, DefaultPagination())
	require.NoError(t, err)
	require.Equal(t, 2, len(result.Bookmarks))
}
func TestQueryBookmarksByTags_Or(t *testing.T) {
	//
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.TagVarietySet())()

	// Linux and GNU both have "linux" or "os"
	result, err := QueryBookmarksByTags(context.Background(), "", []string{"linux", "os"}, TagOr, false, DefaultPagination())
	require.NoError(t, err)
	require.Equal(t, 2, len(result.Bookmarks))
}
func TestQueryBookmarksByTags_WithQuery(t *testing.T) {
	//
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.TagVarietySet())()

	result, err := QueryBookmarksByTags(context.Background(), "Lang", []string{"programming"}, TagAnd, false, DefaultPagination())
	require.NoError(t, err)
	// "Lang" matches Go Lang and Rust Lang (both have programming tag)
	require.Equal(t, 2, len(result.Bookmarks))
}
func TestBookmarksByTag_Match(t *testing.T) {
	//
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.TagVarietySet())()

	result, err := BookmarksByTag(context.Background(), "programming", DefaultPagination())
	require.NoError(t, err)
	require.Equal(t, 3, len(result.Bookmarks))
	require.Equal(t, uint(3), result.Total)
}
func TestBookmarksByTag_NoMatch(t *testing.T) {
	//
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.DefaultSeedSet())()

	result, err := BookmarksByTag(context.Background(), "nonexistent", DefaultPagination())
	require.NoError(t, err)
	require.Equal(t, 0, len(result.Bookmarks))
}
func TestBookmarksByTags_And(t *testing.T) {
	//
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.TagVarietySet())()

	// GNU has both "linux" and "gnu" tags
	result, err := BookmarksByTags(context.Background(), []string{"linux", "gnu"}, TagAnd, DefaultPagination())
	require.NoError(t, err)
	require.Equal(t, 1, len(result.Bookmarks))
	require.Equal(t, "GNU", result.Bookmarks[0].Title)
}
func TestBookmarksByTags_Or(t *testing.T) {
	//
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.TagVarietySet())()

	// Linux (linux, os) and GNU (linux, gnu, os) match "linux" OR "os"
	result, err := BookmarksByTags(context.Background(), []string{"linux", "os"}, TagOr, DefaultPagination())
	require.NoError(t, err)
	require.Equal(t, 2, len(result.Bookmarks))
}
func TestBookmarksByTags_NilPagination(t *testing.T) {
	//
	db, cleanup := newTestDB(t)
	defer cleanup()
	defer seedDB(t, db, fixtures.DefaultSeedSet())()

	_, err := BookmarksByTags(context.Background(), []string{"a"}, TagAnd, nil)
	require.Error(t, err)
}
