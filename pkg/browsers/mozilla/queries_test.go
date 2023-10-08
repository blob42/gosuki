package mozilla

import (
	"testing"

	"github.com/blob42/gosuki/internal/database"
	"github.com/blob42/gosuki/internal/utils"
	"github.com/gchaincl/dotsql"
	_ "github.com/kr/pretty"
	"github.com/swithek/dotsqlx"
)

func Test_loadQueries(t *testing.T) {

	queries := map[string]string{
		"merged-places-bookmarks": "merged_places_bookmarks.sql",
		"recursive-all-bookmarks": "recursive_all_bookmarks.sql",
		"recursive-modified-bookmarks": "recursive_modified_bookmarks.sql",
	}

	loadedQueries := map[string]*dotsqlx.DotSqlx{}

	exists, err := utils.CheckFileExists("testdata/places.sqlite")
	if err != nil {
		t.Fatal(err)
	}

	if !exists {
		t.Fatal("places file does not exist")
	}

	db := database.NewDB("test_places", "testdata/places.sqlite", database.DBTypeFileDSN)
	defer func() {
		err := db.Close()
		if err != nil {
			t.Fatal()
		}
	}()

	_, err = db.Init()
	if err != nil {
		t.Error(err)
	}

	for q, qfile := range queries {
		dot, err := dotsql.LoadFromFile(qfile)
		if err != nil {
			t.Fatal(err)
		}
		dotx := dotsqlx.Wrap(dot)
		_, err = dotx.Raw(q)
		if err != nil {
			t.Fatal(err)
		}
		loadedQueries[q] = dotx
	}

	// Loading of bookmarks and their folders algorithm:
	//     1. [*] execute merged places_bookmarks table query
	//         [*] scan the query into a bookmark_places struct

	//     3- go through bookmarks and
	//         - add tag nodes
	//         - add url nodes

	//         ?- add hierarchy relationship ?
	//            - store folders as hierarchy using a separate tree
	//            - extract folders tree into a flat tag list
	//            - store tag list with appropriate hierarcy info
	//
	//     4- Sync URLIndex to the the buffer DB

	t.Run("Scanning merged-places-bookmarks", func(t *testing.T) {
		queryName := "merged-places-bookmarks"

		dotx, ok := loadedQueries[queryName]
		if !ok {
			t.Fatalf("cannot load query")
		}
		rowsx, err := dotx.Queryx(db.Handle, queryName)
		if err != nil {
			t.Fatal(err)
		}

		for rowsx.Next() {
			var placebk MergedPlaceBookmark

			err = rowsx.StructScan(&placebk)
			if err != nil {
				t.Error(err)
			}
		}

        t.Run("Select bookmarks", func(t *testing.T) {

            var bookmarks []*MergedPlaceBookmark
            err := loadedQueries[queryName].Select(db.Handle, &bookmarks, queryName)
            if err != nil {
                t.Error(err)
            }

            // pretty.Log(bookmarks)
        })
	})


	t.Run("Scanning recursive-all-bookmarks", func(t *testing.T) {
		queryName := "recursive-all-bookmarks"

		dotx, ok := loadedQueries[queryName]
		if !ok {
			t.Fatalf("cannot load query")
		}
		rowsx, err := dotx.Queryx(db.Handle, queryName)
		if err != nil {
			t.Fatal(err)
		}

		for rowsx.Next() {
			var mozBk MozBookmark

			err = rowsx.StructScan(&mozBk)
			if err != nil {
				t.Error(err)
			}
		}

        t.Run("Select bookmarks", func(t *testing.T) {

            var bookmarks []*MozBookmark
            err := loadedQueries[queryName].Select(db.Handle, &bookmarks, queryName)
            if err != nil {
                t.Error(err)
            }

            // pretty.Log(bookmarks)
        })
	})
}
