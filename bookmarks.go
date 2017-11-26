package main

import (
	"strings"
)

// Bookmark type
type Bookmark struct {
	URL      string   `json:"url"`
	Metadata string   `json:"metadata"`
	Tags     []string `json:"tags"`
	Desc     string   `json:"desc"`
	Node     Node
	//flags int
}

func (bk *Bookmark) add(db *DB) {
	//log.Debugf("Adding bookmark %s", bk.url)
	_db := db.handle

	tx, err := _db.Begin()
	logPanic(err)

	stmt, err := tx.Prepare(`INSERT INTO bookmarks(URL, metadata, tags, desc, flags) VALUES (?, ?, ?, ?, ?)`)
	logError(err)
	defer stmt.Close()

	_, err = stmt.Exec(bk.URL, bk.Metadata, strings.Join(bk.Tags, " "), "", 0)
	sqlErrorMsg(err, bk.URL)

	err = tx.Commit()
	logError(err)
}
