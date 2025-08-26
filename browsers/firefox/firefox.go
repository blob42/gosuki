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

package firefox

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/blob42/gosuki/hooks"
	"github.com/blob42/gosuki/internal/database"
	"github.com/blob42/gosuki/pkg/browsers"
	"github.com/blob42/gosuki/pkg/browsers/mozilla"
	"github.com/blob42/gosuki/pkg/events"
	"github.com/blob42/gosuki/pkg/logging"
	"github.com/blob42/gosuki/pkg/modules"
	"github.com/blob42/gosuki/pkg/parsing"
	"github.com/blob42/gosuki/pkg/profiles"

	"github.com/blob42/gosuki/internal/utils"
	"github.com/blob42/gosuki/pkg/tree"
	"github.com/blob42/gosuki/pkg/watch"

	"github.com/fsnotify/fsnotify"
)

var (
	log = logging.GetLogger("ff")
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

	parsing.Counter
	lastSentProgress float64

	// Map from moz_bookmarks tag ids to a tree node
	// tagMap is used as a quick lookup table into the node tree
	tagMap map[string]*tree.Node

	// map from moz_bookmarks folder id to a folder node in the tree
	folderMap map[sqlid]*tree.Node

	// internal folder map used for scanning
	folderScanMap map[sqlid]*MozFolder

	lastRunAt time.Time

	activeProfile *profiles.Profile

	activeFlavour *browsers.BrowserDef
}

// GetCurFlavour implements profiles.ProfileManager.
func (f *Firefox) GetCurFlavour() *browsers.BrowserDef {
	return f.activeFlavour
}

func (firefox *Firefox) ListFlavours() []browsers.BrowserDef {
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

	folderQueryArgs := map[string]any{
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
		f.folderScanMap[folder.ID] = folder
	}

	for _, folder := range folders {
		// Ignore the `tags` virtual folder
		if folder.ID != 4 {
			f.addFolderNode(*folder)
		}
	}

	return folders, err
}

// load bookmarks and tags into the node tree then attach them to
// their assigned folder hierarchy
func (f *Firefox) loadBookmarksToTree(bookmarks []*MozBookmark, runTask bool) {

	for _, bkEntry := range bookmarks {
		// Create/Update URL node and apply tag node
		created, urlNode := f.addURLNode(bkEntry.URL, bkEntry.Title, bkEntry.PlDesc)
		if !created {
			log.Debugf("url <%s> already in url index", bkEntry.URL)
		} else {
			f.IncURLCount()
			progress := f.Progress()
			if progress-f.lastSentProgress >= 0.05 || progress == 1 {
				f.lastSentProgress = progress
				go func() {
					msg := events.ProgressUpdateMsg{
						ID:           f.ModInfo().ID,
						Instance:     f,
						CurrentCount: f.URLCount(),
						Total:        f.Total(),
					}
					if runTask {
						msg.NewBk = true
					}
					events.TUIBus <- msg
				}()
			}
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
				log.Tracef("tag <%s> already in tag map", tagNode.Title)
			}

			// Add tag name to urlnode tags
			urlNode.Tags = utils.Extends(urlNode.Tags, tagNode.Title)

			// Add URL node as child of Tag node
			// Parent will be a folder or nothing?
			tree.AddChild(f.tagMap[tagNode.Title], urlNode)

		}

		// Link this URL node to its corresponding folder node if it exists.
		folderNode, fOk := f.folderMap[bkEntry.ParentID]
		// If we found the parent folder
		if fOk {
			tree.AddChild(folderNode, urlNode)
		}

		err := f.CallHooks(urlNode)
		if err != nil {
			log.Errorf("error calling hooks for <%s>: %s", urlNode.URL, err)
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

	dotx, err := database.DotxQueryEmbedFS(mozilla.EmbeddedSQLQueries, mozilla.MozBookmarkQueryFile)
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

	dotx, err := database.DotxQueryEmbedFS(mozilla.EmbeddedSQLQueries,
		mozilla.MozChangedBookmarkQueryFile)

	if err != nil {
		return nil, err
	}

	queryArgs := map[string]any{
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
		Counter:       &parsing.BrowserCounter{},
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

func (f Firefox) fullID() string {
	return fmt.Sprintf("%s_%s", f.Name, f.Profile)
}

// GetProfiles implements the profiles.ProfileManager interface
func (*Firefox) GetProfiles(flavour string) ([]*profiles.Profile, error) {
	customProfiles := profiles.FromCustom(FFConfig.CustomProfiles, flavour)
	profiles, err := FirefoxProfileManager.GetProfiles(flavour)
	if err != nil {
		return nil, err
	}

	return append(profiles, customProfiles...), nil
}

func (f *Firefox) WatchAllProfiles() bool {
	return FFConfig.WatchAllProfiles
}

func (f *Firefox) UseProfile(p *profiles.Profile, flv *browsers.BrowserDef) error {
	// update instance profile name
	if p != nil {
		f.Profile = p.Name
		f.activeProfile = p

		// setup the bookmark dir
		if bookmarkDir, err := p.AbsolutePath(); err != nil {
			return err
		} else {
			f.BkDir = bookmarkDir
			return nil
		}
	}

	if flv != nil {
		f.activeFlavour = flv
	}
	return nil
}

func (f *Firefox) GetProfile() *profiles.Profile {
	return f.activeProfile
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
	log.Debugf("initializing <%s>", f.fullID())

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
		log.Error(err)
		return modules.ErrWatcherSetup
	}

	if !ok {
		return modules.ErrWatcherSetup
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
// starts. Implements modules.PreLoader interface.
func (f *Firefox) PreLoad(_ *modules.Context) error {
	pc, err := f.initPlacesCopy()
	if err != nil {
		return err
	}

	defer func() {
		if err = pc.Clean(); err != nil {
			log.Errorf("error cleaning tmp places file: %s", err)
		}
	}()

	// load all bookmarks
	start := time.Now()
	bookmarks, err := f.scanBookmarks()
	if err != nil {
		return err
	}
	f.SetTotal(uint(len(bookmarks)))

	// Send total to msg bus
	go func() {
		events.TUIBus <- events.StartedLoadingMsg{
			ID:    modules.ModID(f.Name),
			Total: f.Total(),
		}
	}()

	f.loadBookmarksToTree(bookmarks, false)

	f.SetLastTreeParseRuntime(time.Since(start))
	f.lastRunAt = time.Now().UTC()

	log.Debugf("<%s> parsed %d bookmarks and %d nodes in %s",
		f.fullID(),
		f.URLCount(),
		f.NodeCount(),
		f.LastFullTreeParseRT())

	// Sync the URLIndex to the buffer
	// We do not use the NodeTree here as firefox tags are represented
	// as a flat tree which is not efficient, we use the go hashmap instead

	database.SyncURLIndexToBuffer(f.URLIndexList, f.URLIndex, f.BufferDB)

	if err = f.BufferDB.SyncToCache(); err != nil {
		return err
	}

	// go database.Cache.DB.SyncToDisk(database.GetDBFullPath())
	// schedule a sync to disk
	database.ScheduleBackupToDisk()

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
	ff.loadBookmarksToTree(bookmarks, true)
	// tree.PrintTree(ff.NodeTree)

	//NOTE: we don't rebuild the index from the tree here as the source of
	// truth is the URLIndex and not the tree. The tree is only used for
	// reprensenting the bookmark hierarchy in a conveniant way.

	database.SyncURLIndexToBuffer(ff.URLIndexList, ff.URLIndex, ff.BufferDB)
	ff.BufferDB.SyncTo(database.Cache.DB)
	database.ScheduleBackupToDisk()

	ff.SetLastWatchRuntime(time.Since(startRun))
	ff.lastRunAt = time.Now().UTC()
}

// Implement modules.Shutdowner
func (f *Firefox) Shutdown() error {
	var err error
	if f.places != nil {
		err = f.places.Close()
	}
	return err
}

// and only pass extra details about tag/url along in some data structure
// PROBLEM: tag nodes use IDs and URL nodes use URL as hashes
//
// addURLNode inserts a new URL node into the URL index or updates an existing one if the URL already exists.
// It constructs the module name based on the Firefox instance's active flavour and profile, if present.
// Returns true and the created node if a new URL node was added, or false and the existing node if the URL was already present.
//
// Returns:
//   - created: true if a new node was created, false if an existing node was updated.
//   - node: The tree.Node representing the URL entry, either newly created or existing.
func (f *Firefox) addURLNode(url, title, desc string) (bool, *tree.Node) {

	var urlNode *tree.Node
	var created bool

	iURLNode, exists := f.URLIndex.Get(url)

	modName := f.Name
	if f.activeFlavour != nil {
		modName = fmt.Sprintf("%s_%s", modName, f.activeFlavour.Flavour)
	}
	if f.activeProfile != nil {
		modName = fmt.Sprintf("%s_%s", modName, f.activeProfile.Name)
	}

	if !exists {
		urlNode = &tree.Node{
			Title:  title,
			Type:   tree.URLNode,
			URL:    url,
			Desc:   desc,
			Module: modName, // module which created this node
		}

		log.Tracef("inserting url %s in url index", url)
		f.URLIndex.Insert(url, urlNode)
		f.URLIndexList = append(f.URLIndexList, url)
		f.IncNodeCount()

		created = true

	} else {
		urlNode = iURLNode.(*tree.Node)
		//TEST:
		// update title and desc
		urlNode.Title = title
		urlNode.Desc = desc
	}

	return created, urlNode
}

// adds a new tagNode if it is not yet in the tagMap
// returns true if tag added or false if already existing
// returns the created tagNode
func (f *Firefox) addTagNode(tagName string) (bool, *tree.Node) {
	// Check if "tags" branch exists or create it
	log.Tracef("<%s> adding tag <%s>", f.fullID(), tagName)
	var branchOk bool
	var tagsBranch *tree.Node
	for _, c := range f.NodeTree.Children {
		if c.Title == TagsBranchName {
			branchOk = true
			tagsBranch = c
		}
	}

	if !branchOk {
		tagsBranch = &tree.Node{
			Title: TagsBranchName,
		}
		tree.AddChild(f.NodeTree, tagsBranch)
	}

	// node, exists :=
	node, exists := f.tagMap[tagName]
	if exists {
		return false, node
	}

	tagNode := &tree.Node{
		Title:  tagName,
		Type:   tree.TagNode,
		Parent: f.NodeTree, // root node
	}

	tree.AddChild(tagsBranch, tagNode)
	f.tagMap[tagName] = tagNode
	f.IncNodeCount()

	return true, tagNode
}

// add a folder node to the parsed node tree under the specified folder parent
// returns true if a new folder is created and false if folder already exists
func (f *Firefox) addFolderNode(folder MozFolder) (bool, *tree.Node) {

	// use hashmap.RBTree to keep an index of scanned folders pointing
	// to their corresponding nodes in the tree

	folderNode, seen := f.folderMap[folder.ID]

	if seen {
		// Update folder name if changed

		if folderNode.Title != folder.Title &&
			// Ignore root folders since we use our custom names
			!utils.InList([]int{2, 3, 5, 6}, int(folder.ID)) {
			log.Debugf("folder node <%s> updated to <%s>", folderNode.Title, folder.Title)
			folderNode.Title = folder.Title
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
	if utils.InList([]int{2, 3, 5, 6}, int(folder.ID)) {
		folderNode.Title = mozilla.RootFolderNames[folder.ID]
		tree.AddChild(f.NodeTree, folderNode)
	} else {
		folderNode.Title = folder.Title
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
	f.folderMap[folder.ID] = folderNode
	f.IncNodeCount()

	return true, folderNode
}

// Copies places.sqlite to a tmp dir to read a VFS lock sqlite db
func (f *Firefox) initPlacesCopy() (mozilla.PlaceCopyJob, error) {
	pc := mozilla.NewPlaceCopyJob()

	// Construire le chemin source correctement
	sourcePath := filepath.Join(f.BkDir, f.BkFile)

	// Copier seulement le fichier principal
	err := utils.CopyFileToTmp(sourcePath, pc.Path(), f.BkFile)
	if err != nil {
		return pc, fmt.Errorf("could not copy places.sqlite: %w", err)
	}

	// Construire le chemin destination correctement
	destPath := filepath.Join(pc.Path(), f.BkFile)

	opts := FFConfig.PlacesDSN
	f.places, err = database.NewDB("places", destPath, database.DBTypeFileDSN, opts).Init()
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
var _ modules.PreLoader = (*Firefox)(nil)
var _ modules.Shutdowner = (*Firefox)(nil)
var _ watch.WatchRunner = (*Firefox)(nil)
var _ hooks.HookRunner = (*Firefox)(nil)
var _ parsing.Counter = (*Firefox)(nil)
var _ profiles.ProfileManager = (*Firefox)(nil)
