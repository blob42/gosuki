package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var testBookmarks = []RawBookmark{
	{
		ID:       1,
		URL:      "https://example.com",
		Metadata: "Example Homepage",
		Tags:     ",example,homepage,",
		Desc:     "The example domain homepage",
		Modified: 1620000000,
		Flags:    0,
		Module:   "default",
		XHSum:    "16455977584657621362",
		Version:  1,
	},
	{
		ID:       2,
		URL:      "https://testpage.org",
		Metadata: "Test Page",
		Tags:     ",demo,testing,",
		Desc:     "A sample test page for demonstrations",
		Modified: 1620000001,
		Flags:    0,
		Module:   "default",
		XHSum:    "1083303603766938985",
		Version:  2,
	},
	{
		ID:       3,
		URL:      "https://golang.org",
		Metadata: "Go Language",
		Tags:     ",go,programming,",
		Desc:     "Official Go programming language website",
		Modified: 1620000002,
		Flags:    0,
		Module:   "default",
		XHSum:    "13775917883737511513",
		Version:  3,
	},
	{
		ID:       4,
		URL:      "https://github.com",
		Metadata: "GitHub",
		Tags:     ",code,repository,",
		Desc:     "Hosted version control platform for software development",
		Modified: 1620000003,
		Flags:    0,
		Module:   "default",
		XHSum:    "10075049150054132694",
		Version:  4,
	},
	{
		ID:       5,
		URL:      "https://wikipedia.com",
		Metadata: "Google Docs",
		Tags:     ",knowledge,wiki,",
		Desc:     "Online document creation and collaboration tool",
		Modified: 1620000004,
		Flags:    0,
		Module:   "default",
		XHSum:    "10072264466950902871",
		Version:  5,
	},
}

// This will test tag merging and unique modification changes through 3 levels
// - Buffer -> CacheL1 -> CacheL2
// - CacheL1 and CacheL2 are initialized with a starting dataset
// - Each test will consist of inserting some data in the buffer, then syncing
// the buffer to CacheL1 then syncing CacheL1 to CacheL2
//
// ## Test Cases:
// 1. inserting a bookmark that does not exist in CacheL2 should appear in
// l2 cache
// 2. inserting the exact same bookmark that exist in cachel2  should not change
// l2 cache
// 3. inserting a bookmark with different tags should have cachel2 with a
// different bookmark and its xhsum column changed
func TestSyncToCache(t *testing.T) {
	var count int
	var version uint64

	go startSchedulers()

	Clock = &LamportClock{}
	require.Equal(t, uint64(0), Clock.Value)
	buffer := getBuffer(t)
	cacheL1 := getCache(t, CacheName)
	cacheL2 := getCache(t, L2CacheName)
	require.Equal(t, uint64(5), Clock.Value)

	// if a new bookmark is inserted in the buffer it should appear in l2 cache
	t.Run("Test case 1: New bookmark", func(t *testing.T) {
		bm := Bookmark{
			URL:    "http://example.com/bookmark1",
			Title:  "Test Title",
			Tags:   []string{"tag1"},
			Desc:   "",
			Module: "test",
		}
		sum := xhsum(
			bm.URL,
			bm.Title,
			NewTags(bm.Tags, TagSep).PreSanitize().Sort().String(true),
			bm.Desc,
		)
		err := buffer.UpsertBookmark(&bm)
		require.NoError(t, err)

		buffer.SyncTo(cacheL1)
		cacheL1.SyncTo(cacheL2)

		err = cacheL2.Handle.Get(&count, `SELECT COUNT(*) FROM gskbookmarks WHERE URL = ?`, bm.URL)
		require.NoError(t, err)
		if count != 1 {
			t.Errorf("Expected 1 bookmark in cacheL2 but got %d", count)
		}

		err = cacheL2.Handle.Get(&version, `SELECT COALESCE(max(version),0) from gskbookmarks`, bm.URL)
		require.NoError(t, err)
		require.Equal(t, 6, int(version))

		// saved bookmark
		sbm := RawBookmark{}
		err = cacheL2.Handle.Get(&sbm, `SELECT * FROM gskbookmarks WHERE url = ?`, bm.URL)
		require.NoError(t, err)
		savedTagString := tagsFromString(sbm.Tags, TagSep).Sort()

		if sbm.URL != bm.URL || sbm.Metadata != bm.Title ||
			!reflect.DeepEqual(savedTagString.Get(), bm.Tags) {
			t.Errorf("Bookmark data mismatch: expected %v, got %v", bm, sbm)
		}
		require.Equal(t, sum, sbm.XHSum)
	})

	// if the same bookmark is inserted, l2 cache is unchanged
	t.Run("Test case 2: Same bookmark", func(t *testing.T) {
		bm := Bookmark{
			URL:    testBookmarks[0].URL,
			Title:  testBookmarks[0].Metadata,
			Tags:   tagsFromString(testBookmarks[0].Tags, TagSep).Get(),
			Desc:   testBookmarks[0].Desc,
			Module: testBookmarks[0].Module,
		}

		tagStr := NewTags(bm.Tags, TagSep).PreSanitize().Sort().String(true)

		sum := xhsum(
			bm.URL,
			bm.Title,
			tagStr,
			bm.Desc,
		)
		err := buffer.UpsertBookmark(&bm)
		require.NoError(t, err)

		require.Equal(t, testBookmarks[0].XHSum, sum)

		buffer.SyncTo(cacheL1)
		cacheL1.SyncTo(cacheL2)

		// new bookmark should exist
		err = cacheL2.Handle.Get(&count, `SELECT COUNT(*) FROM gskbookmarks`)
		require.NoError(t, err)
		require.Equal(t, len(testBookmarks)+1, count)

		err = cacheL2.Handle.Get(&version, `SELECT COALESCE(max(version),0) from gskbookmarks`, bm.URL)
		require.NoError(t, err)
		require.Equal(t, 6, int(version))

		var sbm RawBookmark
		err = cacheL2.Handle.Get(&sbm, `SELECT * FROM gskbookmarks WHERE url = ?`, bm.URL)
		require.NoError(t, err)

		savedTagString := tagsFromString(sbm.Tags, TagSep).Sort()
		if sbm.URL != bm.URL || sbm.Metadata != bm.Title ||
			!reflect.DeepEqual(savedTagString.Get(), bm.Tags) {
			t.Errorf("Bookmark data mismatch: expected %v, got %v", bm, sbm)
		}
		require.Equal(t, sum, sbm.XHSum)
	})

	t.Run("Test case 3: Different tags", func(t *testing.T) {
		bm := Bookmark{
			URL:    testBookmarks[0].URL,
			Title:  testBookmarks[0].Metadata,
			Tags:   []string{"tag1", "tag2"},
			Desc:   testBookmarks[0].Desc,
			Module: testBookmarks[0].Module,
		}

		mergedTags := append(bm.Tags, tagsFromString(testBookmarks[0].Tags, TagSep).Get()...)
		tagStr := NewTags(mergedTags, TagSep).PreSanitize().Sort().String(true)

		sum := xhsum(
			bm.URL,
			bm.Title,
			tagStr,
			bm.Desc,
		)
		err := buffer.UpsertBookmark(&bm)
		require.NoError(t, err)
		require.NotEqual(t, testBookmarks[0].XHSum, sum)

		buffer.SyncTo(cacheL1)
		cacheL1.SyncTo(cacheL2)

		// bookmark count should be the same (+1 from previous test)
		err = cacheL2.Handle.Get(&count, `SELECT COUNT(*) FROM gskbookmarks`)
		require.NoError(t, err)
		require.Equal(t, len(testBookmarks)+1, count)

		err = cacheL2.Handle.Get(&version, `SELECT COALESCE(max(version),0) from gskbookmarks`, bm.URL)
		require.NoError(t, err)
		require.Equal(t, 7, int(version))
		require.Equal(t, 7, int(Clock.Value))

		// saved bookmark
		var sbm RawBookmark
		err = cacheL2.Handle.Get(&sbm, `SELECT * FROM gskbookmarks WHERE url = ?`, bm.URL)
		require.NoError(t, err)

		if sbm.URL != bm.URL || sbm.Metadata != bm.Title {
			t.Errorf("Bookmark data mismatch: expected %#v, got %#v", bm, sbm)
		}
		savedTagString := tagsFromString(sbm.Tags, TagSep).Sort()
		require.ElementsMatch(t, mergedTags, savedTagString.Get())
		require.Equal(t, sum, sbm.XHSum)
	})

	buffer.Close()
	cacheL1.Close()
	cacheL2.Close()
}

func TestSyncToDisk(t *testing.T) {
	Clock = &LamportClock{}
	srcDB, dstDB := setupSyncToDiskDBs(t)

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

	for i, bm := range bookmarks {
		require.Equal(t, bm, dstBookmarks[i])
	}
}

func getBuffer(t *testing.T) *DB {
	db, err := NewBuffer("test_buffer")
	require.NoError(t, err)
	return db
}

// cache preloaded with test data
func getCache(t *testing.T, name string) *DB {
	var err error
	clock := Clock.Value

	cache, err := NewDB(name, "", DBTypeCacheDSN).Init()
	require.NoError(t, err)

	err = cache.InitSchema(context.Background())
	require.NoError(t, err)

	tx, err := cache.Handle.Begin()
	require.NoError(t, err)
	for _, rbk := range testBookmarks {
		if name == L2CacheName {
			clock = Clock.LocalTick()
		}
		// fmt.Printf("%s: %s\n", rbk.URL, xhsum(
		// 	rbk.URL,
		// 	rbk.Metadata,
		// 	rbk.Tags,
		// 	rbk.Desc,
		// ))
		_, err = tx.Exec(`INSERT INTO
		gskbookmarks(
			url,
			metadata,
			tags,
			desc,
			module,
			xhsum,
			version
			
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			rbk.URL,
			rbk.Metadata,
			rbk.Tags,
			rbk.Desc,
			rbk.Module,
			xhsum(
				rbk.URL,
				rbk.Metadata,
				rbk.Tags,
				rbk.Desc,
			),
			clock,
		)
		require.NoError(t, err)
	}
	err = tx.Commit()
	require.NoError(t, err)
	if name == L2CacheName {
		require.Equal(t, 5, int(Clock.Value))
	}

	var count int
	err = cache.Handle.Get(&count, `SELECT COUNT(*) FROM gskbookmarks`)
	require.NoError(t, err)
	require.Equal(t, len(testBookmarks), count)

	return cache
}

func setupSyncToDiskDBs(t *testing.T) (*DB, *DB) {
	srcDB := getBuffer(t)

	tmpDir := t.TempDir() // Create a temporary directory for the test database files
	dstPath := filepath.Join(tmpDir, "gosukidb_test.sqlite")
	dstDB := NewDB("test_sync_dst", dstPath, DBTypeFileDSN, DsnOptions{})
	initLocalDB(srcDB, dstPath)

	_, err := dstDB.Init()
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
