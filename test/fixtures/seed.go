// Package fixtures provides seed data and seeding helpers for gosuki tests.
package fixtures

import (
	"fmt"
	"strings"
)

// SeedBookmark defines a bookmark for test seeding with controlled modified
// timestamps. Modified is a unix timestamp used for sort ordering tests.
type SeedBookmark struct {
	URL      string
	Title    string
	Tags     []string
	Desc     string
	Modified int64 // Unix timestamp — controlled for sort testing
	Module   string
}

// DefaultSeedSet returns 5 bookmarks with spaced modified timestamps
// (1000, 2000, 3000, 4000, 5000) for unambiguous sort verification.
//
// Order by modified: Alpha(1000) < Beta(2000) < Gamma(3000) < Delta(4000) < Epsilon(5000)
func DefaultSeedSet() []SeedBookmark {
	return []SeedBookmark{
		{URL: "https://alpha.com", Title: "Alpha", Modified: 1000, Tags: []string{"a"}, Module: "test"},
		{URL: "https://beta.com", Title: "Beta", Modified: 2000, Tags: []string{"b"}, Module: "test"},
		{URL: "https://gamma.com", Title: "Gamma", Modified: 3000, Tags: []string{"g"}, Module: "test"},
		{URL: "https://delta.com", Title: "Delta", Modified: 4000, Tags: []string{"d"}, Module: "test"},
		{URL: "https://epsilon.com", Title: "Epsilon", Modified: 5000, Tags: []string{"e"}, Module: "test"},
	}
}

// TagVarietySet returns bookmarks with overlapping and unique tags for
// tag-based search and filter testing.
func TagVarietySet() []SeedBookmark {
	return []SeedBookmark{
		{URL: "https://go-lang.com", Title: "Go Lang", Modified: 1000, Tags: []string{"go", "programming"}, Module: "test"},
		{URL: "https://rust-lang.com", Title: "Rust Lang", Modified: 2000, Tags: []string{"rust", "programming"}, Module: "test"},
		{URL: "https://python.org", Title: "Python", Modified: 3000, Tags: []string{"python", "programming"}, Module: "test"},
		{URL: "https://linux.org", Title: "Linux", Modified: 4000, Tags: []string{"linux", "os"}, Module: "test"},
		{URL: "https://gnu.org", Title: "GNU", Modified: 5000, Tags: []string{"linux", "gnu", "os"}, Module: "test"},
	}
}

// TagsToDBFormat converts a tag slice to the DB storage format: comma-separated
// with leading and trailing commas, e.g. ",tag1,tag2,"
func TagsToDBFormat(tags []string) string {
	if len(tags) == 0 {
		return ","
	}
	return "," + strings.Join(tags, ",") + ","
}

// BookmarkToInsertSQL generates an INSERT statement for a single bookmark.
// The xhsum is a placeholder — in real usage it's computed from the bookmark content.
func BookmarkToInsertSQL(bm SeedBookmark) string {
	return fmt.Sprintf(
		"INSERT INTO gskbookmarks (URL, metadata, tags, desc, modified, flags, module, xhsum, version) VALUES ('%s', '%s', '%s', '%s', %d, 0, '%s', 'test-xhsum', 1)",
		bm.URL,
		bm.Title,
		TagsToDBFormat(bm.Tags),
		bm.Desc,
		bm.Modified,
		bm.Module,
	)
}

// BookmarksToInsertSQL generates a batch INSERT SQL string for multiple bookmarks.
func BookmarksToInsertSQL(bookmarks []SeedBookmark) string {
	stmts := make([]string, 0, len(bookmarks))
	for _, bm := range bookmarks {
		stmts = append(stmts, BookmarkToInsertSQL(bm))
	}
	return strings.Join(stmts, "; ")
}
