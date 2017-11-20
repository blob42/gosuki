package main

// Bookmark type
type Bookmark struct {
	url      string
	metadata string
	tags     []string
	desc     string
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

	_, err = stmt.Exec(bk.url, bk.metadata, "", "", 0)
	sqlErrorMsg(err, bk.url)

	err = tx.Commit()
	logError(err)
}
