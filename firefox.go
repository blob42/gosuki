package main

import (
	"database/sql"
	"path"
	"time"

	sqlite3 "github.com/mattn/go-sqlite3"
)

var Firefox = BrowserPaths{
	"places.sqlite",
	"/home/spike/.mozilla/firefox/p1rrgord.default/",
}

const (
	MozPlacesRootID       = 1
	MozPlacesTagsRootID   = 4
	MozPlacesMobileRootID = 6
)

type FFBrowser struct {
	BaseBrowser //embedding
	places      *DB
	urlMap      map[string]*Node // Used instead of node tree for syncing to buffer
}

type FFTag struct {
	id    int
	title string
}

func FFPlacesUpdateHook(op int, db string, table string, rowid int64) {
	log.Debug(op)
}

func NewFFBrowser() IBrowser {
	browser := &FFBrowser{}
	browser.name = "firefox"
	browser.bType = TFirefox
	browser.baseDir = Firefox.BookmarkDir
	browser.bkFile = Firefox.BookmarkFile
	browser.useFileWatcher = false
	browser.Stats = &ParserStats{}
	browser.NodeTree = &Node{Name: "root", Parent: nil, Type: "root"}
	browser.urlMap = make(map[string]*Node)

	// sqlite update hook
	sql.Register(DBUpdateMode,
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				log.Warningf("registered connect hook <%s>", DBUpdateMode)
				conn.RegisterUpdateHook(
					func(op int, db string, table string, rowid int64) {
						switch op {
						case sqlite3.SQLITE_UPDATE:
							log.Warning("Notified of insert on db", db, "table", table, "rowid", rowid)
						}

						// Update hook here
						log.Warningf("notified op %s", op)

					})
				return nil
			},
		})

	// Initialize `places.sqlite`
	bookmarkPath := path.Join(browser.baseDir, browser.bkFile)
	browser.places = DB{}.New("Places", bookmarkPath)
	browser.places.engineMode = DBUpdateMode
	browser.places.InitRO()

	// Buffer that lives accross Run() jobs
	browser.InitBuffer()

	/*
	 *Run debouncer to avoid duplicate running of jobs
	 *when a batch of events is received
	 */

	//browser.eventsChan = make(chan fsnotify.Event, EventsChanLen)
	//go debouncer(3000*time.Millisecond, browser.eventsChan, browser)

	return browser
}

func (bw *FFBrowser) Shutdown() {

	log.Debugf("<%s> shutting down ... ", bw.name)

	err := bw.BaseBrowser.Close()
	if err != nil {
		log.Critical(err)
	}

	err = bw.places.Close()
	if err != nil {
		log.Critical(err)
	}
}

func (bw *FFBrowser) Watch() bool {

	log.Debugf("<%s> TODO ... ", bw.name)

	if !bw.isWatching {
		bw.isWatching = true
		return true
	}

	//return false
	return false
}

func (bw *FFBrowser) Load() {
	bw.BaseBrowser.Load()

	// Parse bookmarks to a flat tree (for compatibility with tree system)
	start := time.Now()
	getFFBookmarks(bw)
	bw.Stats.lastParseTime = time.Since(start)

	// Finished parsing
	//go PrintTree(bw.NodeTree) // debugging
	log.Debugf("<%s> parsed %d bookmarks and %d nodes in %s", bw.name,
		bw.Stats.currentUrlCount, bw.Stats.currentNodeCount, bw.Stats.lastParseTime)

	bw.ResetStats()

	// Sync the urlMap to the buffer
	// We do not use the NodeTree here as firefox tags are represented
	// as a flat tree which is not efficient, we use the go hashmap instead
	// The map contains map[url]*Node elements with urls already containing the
	// right tags.

	syncURLMapToBuffer(bw.urlMap, bw.BufferDB)

	bw.BufferDB.SyncTo(CacheDB)
}

func getFFBookmarks(bw *FFBrowser) {

	QGetBookmarks := `WITH bookmarks AS

	(SELECT moz_places.url AS url,
			moz_places.description as desc,
			moz_places.title as urlTitle,
			moz_bookmarks.parent AS tagId
		FROM moz_places LEFT OUTER JOIN moz_bookmarks
		ON moz_places.id = moz_bookmarks.fk
		WHERE moz_bookmarks.parent
		IN (SELECT id FROM moz_bookmarks WHERE parent = ? ))

	SELECT url, IFNULL(urlTitle, ''), IFNULL(desc,''),
			tagId, moz_bookmarks.title AS tagTitle

	FROM bookmarks LEFT OUTER JOIN moz_bookmarks
	ON tagId = moz_bookmarks.id
	ORDER BY url`

	//QGetTags := "SELECT id,title from moz_bookmarks WHERE parent = %d"

	rows, err := bw.places.handle.Query(QGetBookmarks, MozPlacesTagsRootID)
	if err != nil {
		log.Error(err)
	}

	tagMap := make(map[int]*Node)

	// Rebuild node tree
	// Note: the node tree is build only for compatilibity
	// pruposes with tree based bookmark parsing.
	// For efficiency reading after the initial parse from places.sqlite,
	// reading should be done in loops in instead of tree parsing.
	rootNode := bw.NodeTree

	/*
	 *This pass is used only for fetching bookmarks from firefox.
	 *Checking against the URLIndex should not be done here
	 */
	for rows.Next() {
		var url, title, tagTitle, desc string
		var tagId int
		err = rows.Scan(&url, &title, &desc, &tagId, &tagTitle)
		//log.Debugf("%s|%s|%s|%d|%s", url, title, desc, tagId, tagTitle)
		if err != nil {
			log.Error(err)
		}

		/*
		 * If this is the first time we see this tag
		 * add it to the tagMap and create its node
		 */
		tagNode, tagNodeExists := tagMap[tagId]
		if !tagNodeExists {
			// Add the tag as a node
			tagNode = new(Node)
			tagNode.Type = "tag"
			tagNode.Name = tagTitle
			tagNode.Parent = rootNode
			rootNode.Children = append(rootNode.Children, tagNode)
			tagMap[tagId] = tagNode
			bw.Stats.currentNodeCount++
		}

		// Add the url to the tag
		urlNode, urlNodeExists := bw.urlMap[url]
		if !urlNodeExists {
			urlNode = new(Node)
			urlNode.Type = "url"
			urlNode.URL = url
			urlNode.Name = title
			urlNode.Desc = desc
			bw.urlMap[url] = urlNode
		}

		// Add tag to urlnode tags
		urlNode.Tags = append(urlNode.Tags, tagNode.Name)

		// Set tag as parent to urlnode
		urlNode.Parent = tagMap[tagId]

		// Add urlnode as child to tag node
		tagMap[tagId].Children = append(tagMap[tagId].Children, urlNode)

		bw.Stats.currentUrlCount++
		bw.Stats.currentNodeCount++
	}

	/*
	 *Build tags for each url then check against URLIndex
	 *for changes
	 */

	// Check if url already in index TODO: should be done in new pass
	//iVal, found := bw.URLIndex.Get(urlNode.URL)

	/*
	 * The fields where tags may change are hashed together
	 * to detect changes in futre parses
	 * To handle tag changes we need to get all parent nodes
	 *  (tags) for this url then hash their concatenation
	 */

	//nameHash := xxhash.ChecksumString64(urlNode.Name)

}

func (bw *FFBrowser) Run() {

}
