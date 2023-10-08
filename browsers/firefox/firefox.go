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

// TODO: unit test critical error should shutdown the browser
// TODO: shutdown procedure (also close reducer)
// TODO: handle flag management from this package
package firefox

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/blob42/gosuki/hooks"
	"github.com/blob42/gosuki/internal/database"
	"github.com/blob42/gosuki/internal/logging"
	"github.com/blob42/gosuki/pkg/browsers/mozilla"
	"github.com/blob42/gosuki/pkg/modules"
	"github.com/blob42/gosuki/pkg/profiles"

	"github.com/blob42/gosuki/internal/utils"
	"github.com/blob42/gosuki/pkg/tree"
	"github.com/blob42/gosuki/pkg/watch"

	"github.com/fsnotify/fsnotify"
	sqlite3 "github.com/mattn/go-sqlite3"
)

var (
	log            = logging.GetLogger("FF")
)

const (
	WatchMinJobInterval = 1500 * time.Millisecond
	TagsBranchName      = mozilla.TagsBranchName // name of the `tags` branch in the node tree
)

type sqlid = mozilla.Sqlid
type timestamp = int64

type AutoIncr struct {
	ID sqlid
}

type MozBookmark = mozilla.MozBookmark
type MozFolder = mozilla.MozFolder

type Firefox struct {
	*FirefoxConfig

	// sqlite con to places.sqlite
	places *database.DB

	// All elements stored in URLIndex
	URLIndexList []string

	// Map from moz_bookmarks tag ids to a tree node
	// tagMap is used as a quick lookup table into the node tree
	tagMap map[string]*tree.Node

	// map from moz_bookmarks folder id to a folder node in the tree
	folderMap map[sqlid]*tree.Node

	// internal folder map used for scanning
	folderScanMap map[sqlid]*MozFolder

	lastRunAt time.Time
}


func (firefox *Firefox) ListFlavours() []profiles.BrowserFlavour {
	return FirefoxProfileManager.ListFlavours()
}



// func (ff *Firefox) updateModifiedFolders(since timestamp) ([]*MozFolder, error) {
//     // Get list of modified folders
//     var folders = []*MozFolders
//     folderChangeQuery := map[string]interface
//
//     return nil, nil
// }

// scan all folders from moz_bookmarks and load them into the node tree
// takes a timestamp(int64) parameter to select folders based on last modified date
func (f *Firefox) scanFolders(since timestamp) ([]*MozFolder, error) {

	var folders []*MozFolder
	f.folderScanMap = make(map[sqlid]*MozFolder)

	folderQueryArgs := map[string]interface{}{
		"change_since": since,
	}

	boundQuery, args, err := f.places.Handle.BindNamed(mozilla.QFolders, folderQueryArgs)
	if err != nil {
		return nil, err
	}

	err = f.places.Handle.Select(&folders, boundQuery, args...)
	if err != nil {
		return nil, err
	}

	// store all folders in a hashmap for easier tree construction
	for _, folder := range folders {
		f.folderScanMap[folder.Id] = folder
	}

	for _, folder := range folders {
		// Ignore the `tags` virtual folder
		if folder.Id != 4 {
			f.addFolderNode(*folder)
		}
	}

	return folders, err
}

// load bookmarks and tags into the node tree then attach them to
// their assigned folder hierarchy
func (f *Firefox) loadBookmarksToTree(bookmarks []*MozBookmark) {

	for _, bkEntry := range bookmarks {
		// Create/Update URL node and apply tag node
		ok, urlNode := f.addURLNode(bkEntry.Url, bkEntry.Title, bkEntry.PlDesc)
		if !ok {
			log.Debugf("url <%s> already in url index", bkEntry.Url)
		}

		/*
		 * Iterate through bookmark tags and synchronize new tags with
		 * the node tree.
		 */
		for _, tagName := range strings.Split(bkEntry.Tags, ",") {
			if tagName == "" {
				continue
			}
			seen, tagNode := f.addTagNode(tagName)
			if !seen {
				log.Debugf("tag <%s> already in tag map", tagNode.Name)
			}

			// Add tag name to urlnode tags
			urlNode.Tags = utils.Extends(urlNode.Tags, tagNode.Name)

			// Add URL node as child of Tag node
			// Parent will be a folder or nothing?
			tree.AddChild(f.tagMap[tagNode.Name], urlNode)

			f.CurrentURLCount++
		}

		// Link this URL node to its corresponding folder node if it exists.
		//TODO: add all parent folders in the tags list of this url node
		folderNode, fOk := f.folderMap[bkEntry.ParentId]
		// If we found the parent folder
		if fOk {
			tree.AddChild(folderNode, urlNode)
		}

	}
}

// scans bookmarks from places.sqlite and loads them into the node tree
func (f *Firefox) scanBookmarks() ([]*MozBookmark, error) {

	// scan folders and load them into node tree
	_, err := f.scanFolders(0)
	if err != nil {
		return nil, err
	}

	var bookmarks []*MozBookmark

	dotx, err := database.DotxQueryEmbedFS(mozilla.EmbeddedSqlQueries, mozilla.MozBookmarkQueryFile)
	if err != nil {
		return nil, err
	}
	err = dotx.Select(f.places.Handle, &bookmarks, mozilla.MozBookmarkQuery)

	// load bookmarks and tags into the node tree
	// then attach them to their assigned folder hierarchy

	return bookmarks, err
}

func (f *Firefox) scanModifiedBookmarks(since timestamp) ([]*MozBookmark, error) {
	// scan new/modifed folders and load them into node tree
	_, err := f.scanFolders(since)
	// tree.PrintTree(ff.NodeTree)
	if err != nil {
		return nil, err
	}

	var bookmarks []*MozBookmark

	dotx, err := database.DotxQueryEmbedFS(mozilla.EmbeddedSqlQueries,
		mozilla.MozChangedBookmarkQueryFile)

	if err != nil {
		return nil, err
	}

	queryArgs := map[string]interface{}{
		"change_since": since,
	}

	// query, args, err := dotx.NamedQuery(ff.places.Handle, mozilla.MozChangedBookmarkQuery, queryArgs)
	boundQuery, args, err := dotx.BindNamed(f.places.Handle, mozilla.MozChangedBookmarkQuery, queryArgs)
	if err != nil {
		return nil, err
	}

	err = f.places.Handle.Select(&bookmarks, boundQuery, args...)
	if err != nil {
		return nil, err
	}

	return bookmarks, err
}


func NewFirefox() *Firefox {

	return &Firefox{
		FirefoxConfig: FFConfig,
		places:        &database.DB{},
		URLIndexList:  []string{},
		tagMap:        map[string]*tree.Node{},
		folderMap:     map[sqlid]*tree.Node{},
	}
}

func (f Firefox) ModInfo() modules.ModInfo {
	return modules.ModInfo{
		// identify this type of browser
		ID: modules.ModID(f.Name),
		//HACK: duplicate instance with init().RegisterBrowser ??
		New: func() modules.Module {
			return NewFirefox()
		},
	}
}

func (f Firefox) fullId() string {
	return fmt.Sprintf("%s_%s", f.Name, f.Profile)
}

// Implements the profiles.ProfileManager interface
func (*Firefox) GetProfiles(flavour string) ([]*profiles.Profile, error) {
	return FirefoxProfileManager.GetProfiles(flavour)
}

// If should watch all profiles
func (f *Firefox) WatchAllProfiles() bool {
	return FFConfig.WatchAllProfiles
}

// Use custom profile
func (f *Firefox) UseProfile(p profiles.Profile) error {
	// update instance profile name
	f.Profile = p.Name

	// setup the bookmark dir
	if bookmarkDir, err := p.AbsolutePath(); err != nil {
		return err
	} else {
		f.BkDir = bookmarkDir
		return nil
	}
}

func (f *Firefox) Init(ctx *modules.Context, p *profiles.Profile) error {
	if p == nil {
		// setup profile from config
		profile, err := FirefoxProfileManager.GetProfileByName(BrowserName, f.Profile)
		if err != nil {
			return err
		} 
		bookmarkDir, err := profile.AbsolutePath()
		if err != nil {
			return err
		}
		f.BkDir = bookmarkDir
		return f.init(ctx)
	}

	//TEST: try multiple profiles at same time
	// use a new config for this profile
	f.FirefoxConfig = NewFirefoxConfig()
	f.Profile = p.Name


	if bookmarkDir, err := p.AbsolutePath(); err != nil {
		return err
	} else {
		f.BkDir = bookmarkDir
	}

	return f.init(ctx)
}

// TEST:
// Implements browser.Initializer interface
func (f *Firefox) init(ctx *modules.Context) error {
	log.Infof("initializing <%s>", f.fullId())

	watchedPath := f.BkDir
	log.Debugf("Watching path: %s", watchedPath)

	// Setup watcher
	w := &watch.Watch{
		Path:       watchedPath,
		EventTypes: []fsnotify.Op{fsnotify.Write},
		EventNames: []string{filepath.Join(watchedPath, "places.sqlite-wal")},
		ResetWatch: false,
	}

	ok, err := modules.SetupWatchersWithReducer(f.BrowserConfig, modules.ReducerChanLen, w)
	if err != nil {
		return fmt.Errorf("could not setup watcher: %w", err)
	}

	if !ok {
		return errors.New("could not setup watcher")
	}
	

	/*
	 *Run reducer to avoid duplicate jobs when a batch of events is received
	 */

	// Add a reducer to the watcher
	log.Debugf("Running reducer on path <%s>", watchedPath)
	go watch.ReduceEvents(WatchMinJobInterval, f)

	return nil
}

func (f *Firefox) Watch() *watch.WatchDescriptor {
	return f.GetWatcher()
}

func (f Firefox) Config() *modules.BrowserConfig {
	return f.BrowserConfig
}

// Firefox custom logic for preloading the bookmarks when the browser module
// starts. Implements modules.Loader interface.
func (f *Firefox) Load() error {
	pc, err := f.initPlacesCopy()
	if err != nil {
		return err
	}

	defer func(){
		if err := pc.Clean(); err != nil {
			log.Errorf("error cleaning tmp places file: %s", err)
		}
	}()

	// load all bookmarks
	start := time.Now()
	bookmarks, err := f.scanBookmarks()
	if err != nil {
		return err
	}
	f.loadBookmarksToTree(bookmarks)

	f.LastFullTreeParseTime = time.Since(start)
	f.lastRunAt = time.Now().UTC()

	log.Debugf("<%s> parsed %d bookmarks and %d nodes in %s",
		f.fullId(),
		f.CurrentURLCount,
		f.CurrentNodeCount,
		f.LastFullTreeParseTime)
	f.Reset()

	// Sync the URLIndex to the buffer
	// We do not use the NodeTree here as firefox tags are represented
	// as a flat tree which is not efficient, we use the go hashmap instead

	database.SyncURLIndexToBuffer(f.URLIndexList, f.URLIndex, f.BufferDB)

	// Handle empty cache
	if empty, err := database.Cache.DB.IsEmpty(); empty {
		if err != nil {
			return err
		}
		log.Debugf("cache empty: loading buffer to Cachedb")

		f.BufferDB.CopyTo(database.Cache.DB)

		log.Debugf("syncing <%s> to disk", database.Cache.DB.Name)
	} else {
		f.BufferDB.SyncTo(database.Cache.DB)
	}

	// go database.Cache.DB.SyncToDisk(database.GetDBFullPath())
	// schedule a sync to disk
	database.ScheduleSyncToDisk()

	//DEBUG:
	// tree.PrintTree(f.NodeTree)

	// Close the copy places.sqlite
	err = f.places.Close()

	return err
}

// Implements modules.Runner interface
func (ff *Firefox) Run() {
	startRun := time.Now()

	pc, err := ff.initPlacesCopy()
	if err != nil {
		log.Error(err)
	}
	defer pc.Clean()


	// go one step back in time to avoid missing changes
	scanSince := ff.lastRunAt.Add(-1 * time.Second)
	scanSinceSQL := scanSince.UTC().UnixNano() / 1000


	log.Debugf("Checking changes since <%d> %s",
		scanSinceSQL,
		scanSince.Local().Format("Mon Jan 2 15:04:05 MST 2006"))

	bookmarks, err := ff.scanModifiedBookmarks(scanSinceSQL)
	if err != nil {
		log.Error(err)
	}
	ff.loadBookmarksToTree(bookmarks)
	// tree.PrintTree(ff.NodeTree)

	//NOTE: we don't rebuild the index from the tree here as the source of
	// truth is the URLIndex and not the tree. The tree is only used for
	// reprensenting the bookmark hierarchy in a conveniant way.

	database.SyncURLIndexToBuffer(ff.URLIndexList, ff.URLIndex, ff.BufferDB)
	ff.BufferDB.SyncTo(database.Cache.DB)
	database.ScheduleSyncToDisk()


	//TODO!: is LastWatchRunTime alone enough ?
	ff.LastWatchRunTime = time.Since(startRun)
	ff.lastRunAt = time.Now().UTC()
}

// Implement moduls.Shutdowner
func (f *Firefox) Shutdown() error {
	var err error
	if f.places != nil {
		err = f.places.Close()
	}
	return err
}

// TODO: addUrl and addTag share a lot of code, find a way to reuse shared code
// and only pass extra details about tag/url along in some data structure
// PROBLEM: tag nodes use IDs and URL nodes use URL as hashes
func (f *Firefox) addURLNode(url, title, desc string) (bool, *tree.Node) {

	var urlNode *tree.Node
	var created bool

	iURLNode, exists := f.URLIndex.Get(url)
	if !exists {
		urlNode = &tree.Node{
			Name: title,
			Type: tree.URLNode,
			URL:  url,
			Desc: desc,
		}

		log.Debugf("inserting url %s in url index", url)
		f.URLIndex.Insert(url, urlNode)
		f.URLIndexList = append(f.URLIndexList, url)
		f.CurrentNodeCount++

		created = true

	} else {
		urlNode = iURLNode.(*tree.Node)
		//TEST:
		// update title and desc
		urlNode.Name = title
		urlNode.Desc = desc
	}

	// Call hooks
	err := f.CallHooks(urlNode)
	if err != nil {
		log.Errorf("error calling hooks for <%s>: %s", url, err)
	}

	return created, urlNode
}

// adds a new tagNode if it is not yet in the tagMap
// returns true if tag added or false if already existing
// returns the created tagNode
func (f *Firefox) addTagNode(tagName string) (bool, *tree.Node) {
	// Check if "tags" branch exists or create it
	log.Debugf("<%s> adding tag <%s>", f.fullId(), tagName)
	var branchOk bool
	var tagsBranch *tree.Node
	for _, c := range f.NodeTree.Children {
		if c.Name == TagsBranchName {
			branchOk = true
			tagsBranch = c
		}
	}

	if !branchOk {
		tagsBranch = &tree.Node{
			Name: TagsBranchName,
		}
		tree.AddChild(f.NodeTree, tagsBranch)
	}

	// node, exists :=
	node, exists := f.tagMap[tagName]
	if exists {
		return false, node
	}

	tagNode := &tree.Node{
		Name:   tagName,
		Type:   tree.TagNode,
		Parent: f.NodeTree, // root node
	}

	tree.AddChild(tagsBranch, tagNode)
	f.tagMap[tagName] = tagNode
	f.CurrentNodeCount++

	return true, tagNode
}

// add a folder node to the parsed node tree under the specified folder parent
// returns true if a new folder is created and false if folder already exists
func (f *Firefox) addFolderNode(folder MozFolder) (bool, *tree.Node) {

	// use hashmap.RBTree to keep an index of scanned folders pointing
	// to their corresponding nodes in the tree

	folderNode, seen := f.folderMap[folder.Id]

	if seen {
		// Update folder name if changed

		//TODO!: trigger bookmark tag change in gosuki.db
		if folderNode.Name != folder.Title &&
			// Ignore root folders since we use our custom names
			!utils.InList([]int{2, 3, 5, 6}, int(folder.Id)) {
			log.Debugf("folder node <%s> updated to <%s>", folderNode.Name, folder.Title)
			folderNode.Name = folder.Title
		}

		return false, folderNode
	}

	folderNode = &tree.Node{
		// Name: folder.Title,
		Type: tree.FolderNode,
	}

	// keeping the same folder structure as Firefox

	// If this folders' is a Firefox root folder use the appropriate title
	// then add it to the root node
	if utils.InList([]int{2, 3, 5, 6}, int(folder.Id)) {
		folderNode.Name = mozilla.RootFolderNames[folder.Id]
		tree.AddChild(f.NodeTree, folderNode)
	} else {
		folderNode.Name = folder.Title
	}

	// check if folder's parent is already in the tree
	fParent, ok := f.folderMap[folder.Parent]

	// if we already saw folder's parent add it underneath
	if ok {
		tree.AddChild(fParent, folderNode)

		// if we never saw this folders' parent
	} else if folder.Parent != 1 { // recursively build the parent of this folder
		_, ok := f.folderScanMap[folder.Parent]
		if ok {
			_, newParentNode := f.addFolderNode(*f.folderScanMap[folder.Parent])
			tree.AddChild(newParentNode, folderNode)

		}
	}

	// Store a pointer to this folder
	f.folderMap[folder.Id] = folderNode
	f.CurrentNodeCount++

	return true, folderNode
}

// TODO: retire this function after scanBookmarks() is implemented
// load all bookmarks from `places.sqlite` and store them in BaseBrowser.NodeTree
// this method is used the first time gosuki is started or to extract bookmarks
// using a command
func loadBookmarks(f *Firefox) {
	log.Debugf("root tree children len is %d", len(f.NodeTree.Children))
	//QGetTags := "SELECT id,title from moz_bookmarks WHERE parent = %d"
	//

	rows, err := f.places.Handle.Query(mozilla.QgetBookmarks, mozilla.TagsID)
	if err != nil {
		log.Fatal(err)
	}

	// Locked database is critical
	if e, ok := err.(sqlite3.Error); ok {
		if e.Code == sqlite3.ErrBusy {
			log.Critical(err)
			f.Shutdown()
			return
		}
	}
	if err != nil {
		log.Errorf("%s: %s", f.places.Name, err)
		return
	}

	// Rebuilding node tree
	// Note: the node tree is built only for compatilibity with tree based
	// bookmark parsing and might later be useful for debug/UI features.
	// For efficiency reading after the initial Load() from
	// places.sqlite should be done using a loop instad of tree traversal.

	/*
	 *This pass is used only for fetching bookmarks from firefox.
	 *Checking against the URLIndex should not be done here
	 */
	for rows.Next() {
		var url, title, tagTitle, desc string
		var tagID sqlid

		err = rows.Scan(&url, &title, &desc, &tagID, &tagTitle)
		// log.Debugf("%s|%s|%s|%d|%s", url, title, desc, tagId, tagTitle)
		if err != nil {
			log.Error(err)
		}

		/*
		 * If this is the first time we see this tag
		 * add it to the tagMap and create its node
		 */
		ok, tagNode := f.addTagNode(tagTitle)
		if !ok {
			log.Debugf("tag <%s> already in tag map", tagNode.Name)
		}

		// Add the url to the tag
		// NOTE: this call is responsible for updating URLIndexList
		ok, urlNode := f.addURLNode(url, title, desc)
		if !ok {
			log.Debugf("url <%s> already in url index", url)
		}

		// Add tag name to urlnode tags
		urlNode.Tags = append(urlNode.Tags, tagNode.Name)

		// Set tag as parent to urlnode
		tree.AddChild(f.tagMap[tagTitle], urlNode)

		f.CurrentURLCount++
	}

	log.Debugf("root tree children len is %d", len(f.NodeTree.Children))
}

// Copies places.sqlite to a tmp dir to read a VFS lock sqlite db
func (f *Firefox) initPlacesCopy() (mozilla.PlaceCopyJob, error) {
	// create a new copy job
	pc := mozilla.NewPlaceCopyJob()

	err := utils.CopyFilesToTmpFolder(path.Join(f.BkDir, f.BkFile+"*"), pc.Path())
	if err != nil {
		return pc, fmt.Errorf("could not copy places.sqlite to tmp folder: %w", err)
	}

	opts := FFConfig.PlacesDSN

	f.places, err = database.NewDB("places",
		// using the copied places file instead of the original to avoid
		// sqlite vfs lock errors
		path.Join(pc.Path(), f.BkFile),
		database.DBTypeFileDSN, opts).Init()

	if err != nil {
		return pc, err
	}

	return pc, nil
}

// init is required to register the module as a plugin when it is imported
func init() {
	modules.RegisterBrowser(Firefox{FirefoxConfig: FFConfig})


	// Exaple for registering a command under the browser name
	//TIP: cmd.RegisterModCommand(BrowserName, &cli.Command{
	// 	Name: "test",
	// })
	// cmd.RegisterModCommand(BrowserName, &cli.Command{
	// 	Name: "test2",
	// })
}

// interface guards

var _ modules.BrowserModule = (*Firefox)(nil)
var _ modules.ProfileInitializer = (*Firefox)(nil)
var _ profiles.ProfileManager = (*Firefox)(nil)
var _ modules.Loader = (*Firefox)(nil)
var _ modules.Shutdowner = (*Firefox)(nil)
var _ watch.WatchRunner = (*Firefox)(nil)
var _ hooks.HookRunner = (*Firefox)(nil)
var _ watch.Stats = (*Firefox)(nil)
var _ profiles.ProfileManager = (*Firefox)(nil)
