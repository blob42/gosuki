//
// Copyright (c) 2023-2025 Chakib Ben Ziane <contact@blob42.xyz> and [`GoSuki` contributors]
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
	"html"
	"strings"

	"github.com/blob42/gosuki"
	"github.com/blob42/gosuki/internal/utils"
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

// TODO: use context
// Inserts or updates a bookmark in the target database. If a bookmark with the
// same URL already exists due to a constraint, the existing entry is updated
// with the new data.
// NOTE: We don't use sql UPSERT as we need to do a manual merge of some columns
// such as `tags`.
func (db *DB) UpsertBookmark(bk *Bookmark) error {

	var sqlite3Err sqlite3.Error
	var isSqlite3Err bool
	var scannedTags string

	_db := db.Handle

	// Try an INSERT at first
	tryInsertBk, err := _db.Preparex(
		`INSERT INTO
			gskbookmarks(
				URL,
				metadata,
				tags,
				desc,
				flags,
				module,
				xhsum
			)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		log.Errorf("%s: %s", err, bk.URL)
		return err
	}
	defer cleanup(tryInsertBk.Close)

	// UPDATE on constraint errors
	updateBk, err := _db.Preparex(
		`UPDATE gskbookmarks
		SET
			metadata = CASE WHEN ? != '' THEN ? ELSE metadata END,
			desc = CASE WHEN ? != '' THEN ? ELSE desc END,
			tags=?,
			modified=strftime('%s'),
			xhsum=?
		WHERE url=?`,
	)
	defer cleanup(updateBk.Close)
	if err != nil {
		log.Errorf("%s: %s", err, bk.URL)
		return err
	}

	// Stmt to fetch existing bookmark and tags in db
	getTagsStmt, err := _db.Preparex(`SELECT tags FROM gskbookmarks WHERE url=? LIMIT 1`)
	defer cleanup(getTagsStmt.Close)
	if err != nil {
		log.Errorf("%s: %s", err, bk.URL)
		return err
	}

	// Begin transaction
	tx, err := _db.Beginx()
	if err != nil {
		log.Error(err)
		return err
	}

	// sanitize urls
	bk.URL = html.UnescapeString(bk.URL)

	// unescape unicode
	bk.Title = utils.DecodeUnicodeEscapes(bk.Title)
	bk.Desc = utils.DecodeUnicodeEscapes(bk.Desc)

	// sanitize tags
	// avoid using the delim in the query
	// ex: [ "tag,1", "t,g2", "tag3" ] -> [ "tag--1", "t--g2", "tag3" ]
	tags := NewTags(bk.Tags, TagSep).PreSanitize().Sort()

	tagListText := tags.String(true)

	// First try to insert the bookmark (assume it's new)
	// log.Debugf("INSERT INTO gskbookmarks(URL, metadata, tags, desc, flags) VALUES (%s, %s, %s, %s, %d)",
	// 	bk.URL, bk.Metadata, tagListText, "", 0)

	_, err = tx.Stmtx(tryInsertBk).Exec(
		bk.URL,
		bk.Title, //metadata
		tagListText,
		bk.Desc,
		0,         // flags
		bk.Module, // source module that created this mark

		// xhash sum
		xhsum(bk.URL, bk.Title, tagListText, bk.Desc),
	)

	if err != nil {
		sqlite3Err, isSqlite3Err = err.(sqlite3.Error)
		if !isSqlite3Err {
			log.Errorf("%s: %s", err, bk.URL)
			return err
		}
	}

	if err != nil && isSqlite3Err && sqlite3Err.Code != sqlite3.ErrConstraint {
		log.Errorf("%s: %s", err, bk.URL)
		return err
	}

	// We will handle ErrConstraint: only against URL for now. UPDATE the
	// bookmark instead IF xhash(url+metadata+tags+desc) changed
	if err != nil && sqlite3Err.Code == sqlite3.ErrConstraint {
		log.Debugf("Updating bookmark %s", bk.URL)

		// Get existing xhashsum of bookmark
		var targetXHSum string
		err = tx.QueryRowx("SELECT xhsum FROM gskbookmarks WHERE url = ?", bk.URL).Scan(&targetXHSum)
		if err != nil {
			log.Error("%s", err, "url", bk.URL)
			return err
		}

		// We will only update the bookmark if the xhsum changed
		if targetXHSum == xhsum(bk.URL, bk.Title, tagListText, bk.Desc) {
			log.Debug("upsert: same hash skipping", "url", bk.URL)
			return tx.Rollback()
		}

		// First get existing tags for this bookmark if any
		res := tx.Stmtx(getTagsStmt).QueryRow(bk.URL)
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

		_, err = tx.Stmtx(updateBk).Exec(
			bk.Title,
			bk.Title,
			bk.Desc,
			bk.Desc,
			tagListText,

			// xhsum
			xhsum(bk.URL, bk.Title, tagListText, bk.Desc),

			// where clause
			bk.URL,
		)

		if err != nil {
			log.Errorf("%s: %s", err, bk.URL)
			return err
		}
	}

	return tx.Commit()
}

// Inserts a bookmarks to DB. In case of conflict follow the default rules which
// for sqlite is a fail with the error `sqlite3.ErrConstraint`
// DEAD:
func (db *DB) InsertBookmark(bk *Bookmark) {
	//log.Debugf("Adding bookmark %s", bk.URL)
	_db := db.Handle
	tx, err := _db.Beginx()
	if err != nil {
		log.Error(err)
	}

	stmt, err := tx.Preparex(`INSERT INTO gskbookmarks(URL, metadata, tags, desc, flags) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		log.Error(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(bk.URL, bk.Title, strings.Join(bk.Tags, TagSep), "", 0)
	if err != nil {
		log.Errorf("%s: %s", err, bk.URL)
	}

	err = tx.Commit()
	if err != nil {
		log.Error(err)
	}
}
