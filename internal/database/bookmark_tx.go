//
// Copyright ⓒ 2023 Chakib Ben Ziane <contact@blob42.xyz> and [`GoSuki` contributors]
// (https://github.com/blob42/gosuki/graphs/contributors).
//
// All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This file is part of GoSuki.
//
// GoSuki is free software: you can redistribute it and/or modify it under the terms of
// the GNU Affero General Public License as published by the Free Software Foundation,
// either version 3 of the License, or (at your option) any later version.
//
// GoSuki is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
// without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR
// PURPOSE.  See the GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License along with
// gosuki.  If not, see <http://www.gnu.org/licenses/>.

package database

import (
	"strings"

	"github.com/blob42/gosuki"
	sqlite3 "github.com/mattn/go-sqlite3"
)

// Default separator used to join tags in the DB
const TagSep = ","

type Bookmark = gosuki.Bookmark

func cleanup(f func() error) {
	if err := f(); err != nil {
		log.Error(err)
	}
}

// Inserts or updates a bookmarks to the passed DB
// In case of a conflict for a UNIQUE URL constraint,
// update the existing bookmark
func (db *DB) UpsertBookmark(bk *Bookmark) {

	var sqlite3Err sqlite3.Error
	var isSqlite3Err bool
	var scannedTags string

	_db := db.Handle

	//TODO: use UPSERT stmt
	// Prepare statement that does a pure insert only
	tryInsertBk, err := _db.Prepare(
		`INSERT INTO bookmarks(URL, metadata, tags, desc, flags)
			VALUES (?, ?, ?, ?, ?)`,
	)
	if err != nil {
		log.Errorf("%s: %s", err, bk.URL)
	}
	defer cleanup(tryInsertBk.Close)

	updateBk, err := _db.Prepare(
		`UPDATE bookmarks SET metadata=?, tags=?, modified=strftime('%s')
		WHERE url=?`,
	)
	defer cleanup(updateBk.Close)
	if err != nil {
		log.Errorf("%s: %s", err, bk.URL)
	}

	// Stmt to fetch existing bookmark and tags in db
	getTagsStmt, err := _db.Prepare(`SELECT tags FROM bookmarks WHERE url=? LIMIT 1`)
	defer cleanup(getTagsStmt.Close)
	if err != nil {
		log.Errorf("%s: %s", err, bk.URL)
	}

	// Begin transaction
	tx, err := _db.Begin()
	if err != nil {
		log.Error(err)
	}

	// avoids using the delim in the query
	// ex: [ "tag,1", "t,g2", "tag3" ] -> [ "tag--1", "t--g2", "tag3" ]

	tags := NewTags(bk.Tags, TagSep).PreSanitize()

    tagListText := tags.String(true)
	// log.Debugf("inserting tags %#v", tagListText)

	// First try to insert the bookmark (assume it's new)
	// log.Debugf("INSERT INTO bookmarks(URL, metadata, tags, desc, flags) VALUES (%s, %s, %s, %s, %d)",
	// 	bk.URL, bk.Metadata, tagListText, "", 0)

	_, err = tx.Stmt(tryInsertBk).Exec(
		bk.URL,
		bk.Metadata,
		tagListText,
		"", // desc
		0, // flags
	)

	if err != nil {
		sqlite3Err, isSqlite3Err = err.(sqlite3.Error)
		if !isSqlite3Err {
			log.Errorf("%s: %s", err, bk.URL)
		}
	}

	if err != nil && isSqlite3Err && sqlite3Err.Code != sqlite3.ErrConstraint {
		log.Errorf("%s: %s", err, bk.URL)
	}
	// We will handle ErrConstraint ourselves

	// ErrConstraint means the bookmark (url) already exists in table,
	// we need to update it instead.
	if err != nil && sqlite3Err.Code == sqlite3.ErrConstraint {
		log.Debugf("Updating bookmark %s", bk.URL)

		// First get existing tags for this bookmark if any ?
		res := tx.Stmt(getTagsStmt).QueryRow(
			bk.URL,
		)
		res.Scan(&scannedTags)
		cacheTags := TagsFromString(scannedTags, TagSep)

		// If tags are different, merge current bookmark tags and existing tags
		// Put them in a map first to remove duplicates
		tagMap := make(map[string]bool)
		for _, v := range cacheTags.tags {
			tagMap[v] = true
		}
		for _, v := range bk.Tags {
			tagMap[v] = true
		}

		newTags := Tags{delim: TagSep} // merged tags

		// Merge in a single slice
		for k := range tagMap {
			newTags.Add(k)
		}

		tagListText := newTags.StringWrap()
		log.Debugf("Updating bookmark %s with tags <%s>", bk.URL, tagListText)
		_, err = tx.Stmt(updateBk).Exec(
			bk.Metadata,
			tagListText,
			bk.URL,
		)

		if err != nil {
			log.Errorf("%s: %s", err, bk.URL)
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Error(err)
	}
}

// Inserts a bookmarks to the passed DB
// In case of conflict follow the default rules
// which for sqlite is a fail with the error `sqlite3.ErrConstraint`
func (db *DB) InsertBookmark(bk *Bookmark) {
	//log.Debugf("Adding bookmark %s", bk.URL)
	_db := db.Handle
	tx, err := _db.Begin()
	if err != nil {
		log.Error(err)
	}

	stmt, err := tx.Prepare(`INSERT INTO bookmarks(URL, metadata, tags, desc, flags) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		log.Error(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(bk.URL, bk.Metadata, strings.Join(bk.Tags, TagSep), "", 0)
	if err != nil {
		log.Errorf("%s: %s", err, bk.URL)
	}

	err = tx.Commit()
	if err != nil {
		log.Error(err)
	}
}

// CleanTags sanitize the tags strings stored in the gosuki database.
// The input is a string of tags separated by [TagJoinSep].
// func CleanTags(tags string) string {
// 	
// }
