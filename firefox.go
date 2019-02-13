//TODO: unit test critical error should shutdown the browser
//TODO: shutdown procedure (also close reducer)
package main

import (
	"database/sql"
	"gomark/database"
	"gomark/mozilla"
	"gomark/parsing"
	"gomark/tree"
	"gomark/utils"
	"gomark/watch"
	"path"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jmoiron/sqlx"
	sqlite3 "github.com/mattn/go-sqlite3"
)

const (
	QGetBookmarkPlace = `
	SELECT *
	FROM moz_places
	WHERE id = ?
	`
	QBookmarksChanged = `
	SELECT id,type,IFNULL(fk, -1),parent,IFNULL(title, '') from moz_bookmarks
	WHERE(lastModified > :last_runtime_utc
		AND lastModified < strftime('%s', 'now') * 1000 * 1000
		AND NOT id IN (:not_root_tags)
		)
	`

	QGetBookmarks = `WITH bookmarks AS
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
)

var Firefox = BrowserPaths{
	BookmarkFile: mozilla.BookmarkFile,
	BookmarkDir:  mozilla.BookmarkDir,
}

const (
	MozMinJobInterval = 500 * time.Millisecond
)

type FFBrowser struct {
	BaseBrowser  //embedding
	places       *database.DB
	URLIndexList []string // All elements stored in URLIndex
	tagMap       map[sqlid]*tree.Node
	lastRunTime  time.Time
}

const (
	_ = iota
	BkTypeURL
	BkTypeTagFolder
)

type sqlid uint64

const (
	_ = iota
	ffBkRoot
	ffBkMenu
	ffBkToolbar
	ffBkTags
	ffBkOther
	ffBkMobile
)

type AutoIncr struct {
	ID sqlid
}

type FFPlace struct {
	URL         string
	Description sql.NullString
	Title       sql.NullString
	AutoIncr
}

//type FFBookmark struct {
//BType  int `db:type`
//Parent sqlid
//FK     sql.NullInt64
//Title  sql.NullString
//AutoIncr
//}

type FFBookmark struct {
	btype  int `db:type`
	parent sqlid
	fk     sqlid
	title  string
	id     sqlid
}

func FFPlacesUpdateHook(op int, db string, table string, rowid int64) {
	fflog.Debug(op)
}

//TODO: Test browser creation errors
// In case of critical errors degrade the browser to only log errors and disable
// all directives
func NewFFBrowser() IBrowser {
	var err error

	browser := new(FFBrowser)
	browser.name = "firefox"
	browser.bType = TFirefox
	browser.baseDir = Firefox.BookmarkDir
	browser.bkFile = Firefox.BookmarkFile
	browser.useFileWatcher = true
	browser.Stats = &parsing.Stats{}
	browser.NodeTree = &tree.Node{Name: "root", Parent: nil, Type: "root"}
	browser.tagMap = make(map[sqlid]*tree.Node)

	// Initialize `places.sqlite`
	bookmarkPath := path.Join(browser.baseDir, browser.bkFile)

	opts := database.DsnOptions{
		"_journal_mode": "WAL",
	}

	browser.places, err = database.New("places",
		bookmarkPath,
		database.DBTypeFileDSN, opts).Init()
	if err != nil {

		//Check Lock Error
		if err == database.ErrVfsLocked {
			// Try to unlock db
			e := mozilla.UnlockPlaces(browser.baseDir)
			if e != nil {
				log.Panic(e)
			}
		} else {
			log.Panic(err)
		}
	}

	// Buffer that lives accross Run() jobs
	browser.InitBuffer()

	// Setup watcher

	expandedBaseDir, err := filepath.EvalSymlinks(browser.baseDir)

	if err != nil {
		log.Critical(err)
	}

	w := &Watch{
		Path:       expandedBaseDir,
		EventTypes: []fsnotify.Op{fsnotify.Write},
		EventNames: []string{filepath.Join(expandedBaseDir, "places.sqlite-wal")},
		ResetWatch: false,
	}

	browser.SetupFileWatcher(w)

	/*
	 *Run reducer to avoid duplicate running of jobs
	 *when a batch of events is received
	 */

	browser.eventsChan = make(chan fsnotify.Event, EventsChanLen)

	go utils.ReduceEvents(MozMinJobInterval, browser.eventsChan, browser)

	//
	//
	//
	//
	//

	return browser
}

func (bw *FFBrowser) Shutdown() {

	err := bw.places.Close()
	if err != nil {
		fflog.Critical(err)
	}

	fflog.Debugf("shutting down ... ")
	bw.BaseBrowser.Shutdown()
}

func (bw *FFBrowser) Watch() bool {

	if !bw.isWatching {
		go watch.WatcherThread(bw)
		bw.isWatching = true
		fflog.Infof("Watching %s", bw.GetBookmarksPath())
		return true
	}

	return false
}

func (bw *FFBrowser) Load() {
	bw.BaseBrowser.Load()

	// Parse bookmarks to a flat tree (for compatibility with tree system)
	start := time.Now()
	getFFBookmarks(bw)
	bw.Stats.LastFullTreeParseTime = time.Since(start)
	bw.lastRunTime = time.Now().UTC()

	// Finished parsing
	//go PrintTree(bw.NodeTree) // debugging
	fflog.Debugf("parsed %d bookmarks and %d nodes in %s",
		bw.Stats.CurrentUrlCount,
		bw.Stats.CurrentNodeCount,
		bw.Stats.LastFullTreeParseTime)
	bw.ResetStats()

	// Sync the URLIndex to the buffer
	// We do not use the NodeTree here as firefox tags are represented
	// as a flat tree which is not efficient, we use the go hashmap instead

	database.SyncURLIndexToBuffer(bw.URLIndexList, bw.URLIndex, bw.BufferDB)

	// Handle empty cache
	if empty, err := CacheDB.IsEmpty(); empty {
		if err != nil {
			fflog.Error(err)
		}
		fflog.Info("cache empty: loading buffer to Cachedb")

		bw.BufferDB.CopyTo(CacheDB)

		fflog.Debugf("syncing <%s> to disk", CacheDB.Name)
	} else {
		bw.BufferDB.SyncTo(CacheDB)
	}
	go CacheDB.SyncToDisk(database.GetDBFullPath())
}

func getFFBookmarks(bw *FFBrowser) {

	//QGetTags := "SELECT id,title from moz_bookmarks WHERE parent = %d"
	//

	rows, err := bw.places.Handle.Query(QGetBookmarks, ffBkTags)
	log.Debugf("%#v", err)

	// Locked database is critical
	if e, ok := err.(sqlite3.Error); ok {
		if e.Code == sqlite3.ErrBusy {
			fflog.Critical(err)
			bw.Shutdown()
			return
		}
	}
	if err != nil {
		fflog.Errorf("%s: %s", bw.places.Name, err)
		return
	}

	// Rebuild node tree
	// Note: the node tree is build only for compatilibity with tree based
	// bookmark parsing.  For efficiency reading after the initial Load() from
	// places.sqlite should be done using a loop instad of tree traversal.
	rootNode := bw.NodeTree

	/*
	 *This pass is used only for fetching bookmarks from firefox.
	 *Checking against the URLIndex should not be done here
	 */
	for rows.Next() {
		var url, title, tagTitle, desc string
		var tagId sqlid
		err = rows.Scan(&url, &title, &desc, &tagId, &tagTitle)
		//fflog.Debugf("%s|%s|%s|%d|%s", url, title, desc, tagId, tagTitle)
		if err != nil {
			fflog.Error(err)
		}

		/*
		 * If this is the first time we see this tag
		 * add it to the tagMap and create its node
		 */
		tagNode, tagNodeExists := bw.tagMap[tagId]
		if !tagNodeExists {
			// Add the tag as a node
			tagNode = new(tree.Node)
			tagNode.Type = "tag"
			tagNode.Name = tagTitle
			tagNode.Parent = rootNode
			rootNode.Children = append(rootNode.Children, tagNode)
			bw.tagMap[tagId] = tagNode
			bw.Stats.CurrentNodeCount++
		}

		// Add the url to the tag
		var urlNode *tree.Node
		iUrlNode, urlNodeExists := bw.URLIndex.Get(url)
		if !urlNodeExists {
			urlNode = new(tree.Node)
			urlNode.Type = "url"
			urlNode.URL = url
			urlNode.Name = title
			urlNode.Desc = desc
			bw.URLIndex.Insert(url, urlNode)
			bw.URLIndexList = append(bw.URLIndexList, url)

		} else {
			urlNode = iUrlNode.(*tree.Node)
		}

		// Add tag to urlnode tags
		urlNode.Tags = append(urlNode.Tags, tagNode.Name)

		// Set tag as parent to urlnode
		urlNode.Parent = bw.tagMap[tagId]

		// Add urlnode as child to tag node
		bw.tagMap[tagId].Children = append(bw.tagMap[tagId].Children, urlNode)

		bw.Stats.CurrentUrlCount++
	}

}

func (bw *FFBrowser) fetchUrlChanges(rows *sql.Rows,
	bookmarks map[sqlid]*FFBookmark,
	places map[sqlid]*FFPlace) {

	bk := new(FFBookmark)

	// Get the URL that changed
	rows.Scan(&bk.id, &bk.btype, &bk.fk, &bk.parent, &bk.title)

	// We found URL change, urls are specified by
	// type == 1
	// fk -> id of url in moz_places
	// parent == tag id
	//
	// Each tag on a url generates 2 or 3 entries in moz_bookmarks
	// 1. If not existing, a (type==2) entry for the tag itself
	// 2. A (type==1) entry for the bookmakred url with (fk -> moz_places.url)
	// 3. A (type==1) (fk-> moz_places.url) (parent == idOf(tag))

	if bk.btype == BkTypeURL {
		var place FFPlace
		bw.places.Handle.QueryRowx(QGetBookmarkPlace, bk.fk).StructScan(&place)
		fflog.Debugf("Changed URL: %s", place.URL)

		// put url in the places map
		places[place.ID] = &place
	}

	// This is the tag link
	if bk.btype == BkTypeURL &&
		bk.parent > ffBkMobile {

		bookmarks[bk.id] = bk
	}

	// Tags are specified by:
	// type == 2
	// parent == (Id of root )

	if bk.btype == BkTypeTagFolder {
		bookmarks[bk.id] = bk
	}

	for rows.Next() {
		bw.fetchUrlChanges(rows, bookmarks, places)
	}
}

func (bw *FFBrowser) Run() {

	//TODO: Watching is broken. Try to open a new connection on each
	//  watch event

	startRun := time.Now()
	//fflog.Debugf("Checking changes since %s",
	//bw.lastRunTime.Local().Format("Mon Jan 2 15:04:05 MST 2006"))

	queryArgs := map[string]interface{}{
		"not_root_tags":    []int{ffBkRoot, ffBkTags},
		"last_runtime_utc": bw.lastRunTime.UTC().UnixNano() / 1000,
	}

	query, args, err := sqlx.Named(
		QBookmarksChanged,
		queryArgs,
	)
	if err != nil {
		fflog.Error(err)
	}

	query, args, err = sqlx.In(query, args...)
	if err != nil {
		fflog.Error(err)
	}

	query = bw.places.Handle.Rebind(query)
	rows, err := bw.places.Handle.Query(query, args...)
	defer rows.Close()

	if err != nil {
		fflog.Error(err)
	}

	// Found new results in places db since last time we had changes
	//database.DebugPrintRows(rows)
	if rows.Next() {
		changedURLS := make([]string, 0)
		bw.lastRunTime = time.Now().UTC()

		//fflog.Debugf("CHANGE ! Time: %s",
		//bw.lastRunTime.Local().Format("Mon Jan 2 15:04:05 MST 2006"))

		bookmarks := make(map[sqlid]*FFBookmark)
		places := make(map[sqlid]*FFPlace)

		// Fetch all changes into bookmarks and places maps
		bw.fetchUrlChanges(rows, bookmarks, places)

		// For each url
		for urlId, place := range places {
			var urlNode *tree.Node
			changedURLS = utils.Extends(changedURLS, place.URL)
			iUrlNode, urlNodeExists := bw.URLIndex.Get(place.URL)
			if !urlNodeExists {
				urlNode = new(tree.Node)
				urlNode.Type = "url"
				urlNode.URL = place.URL
				urlNode.Name = place.Title.String
				urlNode.Desc = place.Description.String
				bw.URLIndex.Insert(place.URL, urlNode)

			} else {
				urlNode = iUrlNode.(*tree.Node)
			}

			// First get any new tags
			for bkId, bk := range bookmarks {
				if bk.btype == BkTypeTagFolder &&
					// Ignore root direcotires
					bk.btype != ffBkTags {

					tagNode, tagNodeExists := bw.tagMap[bkId]
					if !tagNodeExists {
						tagNode = new(tree.Node)
						tagNode.Type = "tag"
						tagNode.Name = bk.title
						tagNode.Parent = bw.NodeTree
						bw.NodeTree.Children = append(bw.NodeTree.Children,
							tagNode)
						fflog.Debugf("New tag node %s", tagNode.Name)
						bw.tagMap[bkId] = tagNode
					}
				}
			}

			// link tags to urls
			for _, bk := range bookmarks {

				// This effectively applies the tag to the URL
				// The tag link should have a parent over 6 and fk->urlId
				fflog.Debugf("Bookmark parent %d", bk.parent)
				if bk.fk == urlId &&
					bk.parent > ffBkMobile {

					// The tag node should have already been created
					tagNode, tagNodeExists := bw.tagMap[bk.parent]
					if tagNodeExists && urlNode != nil {
						//fflog.Debugf("URL has tag %s", tagNode.Name)

						urlNode.Tags = utils.Extends(urlNode.Tags, tagNode.Name)

						urlNode.Parent = bw.tagMap[bk.parent]
						tree.Insert(bw.tagMap[bk.parent].Children, urlNode)

						bw.Stats.CurrentUrlCount++
					}
				}
			}

		}

		database.SyncURLIndexToBuffer(changedURLS, bw.URLIndex, bw.BufferDB)
		bw.BufferDB.SyncTo(CacheDB)
		CacheDB.SyncToDisk(database.GetDBFullPath())

	}

	//TODO: change logger for more granular debugging

	bw.Stats.LastWatchRunTime = time.Since(startRun)
	//fflog.Debugf("execution time %s", time.Since(startRun))
}
