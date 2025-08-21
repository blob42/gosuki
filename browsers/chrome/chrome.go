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

// Chrome browser module.
//
// Chrome bookmarks are stored in a json file normally called Bookmarks.
// The bookmarks file is updated atomically by chrome for each change to the
// bookmark entries by the user.
//
// Changes are detected by watching the parent directory for fsnotify.Create
// events on the bookmark file. On linux this is done by using fsnotify.
package chrome

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/OneOfOne/xxhash"
	"github.com/fsnotify/fsnotify"

	"github.com/buger/jsonparser"

	"github.com/blob42/gosuki/hooks"
	"github.com/blob42/gosuki/internal/database"
	"github.com/blob42/gosuki/pkg/browsers"
	"github.com/blob42/gosuki/pkg/events"
	"github.com/blob42/gosuki/pkg/logging"
	"github.com/blob42/gosuki/pkg/modules"
	"github.com/blob42/gosuki/pkg/parsing"
	"github.com/blob42/gosuki/pkg/profiles"
	"github.com/blob42/gosuki/pkg/tree"
	"github.com/blob42/gosuki/pkg/watch"
)

var (
	log = logging.GetLogger("chrome")
)

// Type of the func used recursively on each json node entry by [jsonparser.ArrayEach]
type ParseChildJSONFunc func([]byte, jsonparser.ValueType, int, error)
type RecursiveParseJSONFunc func([]byte, []byte, jsonparser.ValueType, int) error

var jsonNodeTypes = map[string]tree.NodeType{
	"folder": tree.FolderNode,
	"url":    tree.URLNode,
}

var jsonNodePaths = struct {
	Type, Children, URL string
}{"type", "children", "url"}

// type used to store json nodes in memory for parsing.
type RawNode struct {
	title        []byte
	nType        []byte
	url          []byte
	children     []byte
	childrenType jsonparser.ValueType
}

func (rawNode *RawNode) parseItems(nodeData []byte) {

	// Paths to lookup in node payload
	paths := [][]string{
		{"type"},
		{"name"}, // Title of page
		{"url"},
		{"children"},
	}

	jsonparser.EachKey(nodeData, func(idx int, value []byte, vt jsonparser.ValueType, err error) {
		if err != nil {
			log.Error("error parsing node items")
		}

		switch idx {
		case 0:
			rawNode.nType = value

		case 1: // name or title
			//currentNode.Name = s(value)
			rawNode.title = value
		case 2:
			//currentNode.URL = s(value)
			rawNode.url = value
		case 3:
			rawNode.children, rawNode.childrenType = value, vt
		}
	}, paths...)
}

// Returns *tree.Node from *RawNode
func (rawNode *RawNode) getNode(ch *Chrome) *tree.Node {
	node := new(tree.Node)
	nType, ok := jsonNodeTypes[string(rawNode.nType)]
	if !ok {
		log.Errorf("unknown node type: %s", rawNode.nType)
	}
	node.Type = nType

	node.Title = string(rawNode.title)
	modName := ch.Name

	if ch.activeFlavour != nil {
		modName = fmt.Sprintf("%s_%s", modName, ch.activeFlavour.Flavour)
	}
	if ch.activeProfile != nil {
		modName = fmt.Sprintf("%s_%s", modName, ch.activeProfile.Name)
	}

	node.Module = modName

	return node
}

// Chrome browser module
type Chrome struct {
	// holds browsers.BrowserConfig
	*ChromeConfig

	parsing.Counter
	lastSentProgress float64

	activeProfile *profiles.Profile

	activeFlavour *browsers.BrowserDef
}

func (ch *Chrome) Init(ctx *modules.Context, p *profiles.Profile) error {
	// NOTE: if called without profile setup default profile
	var err error
	var profile *profiles.Profile

	if p == nil {
		profileName := ch.Profile

		// parse flavour
		parsedProfile := strings.SplitN(profileName, ":", 2)
		if len(parsedProfile) > 1 {
			flavour := parsedProfile[0]
			profileName = parsedProfile[1]
			profile, err = ProfileManager.GetProfileByID(flavour, profileName)
		} else {
			profile, err = ProfileManager.GetProfileByID(BrowserName, ch.Profile)
		}

		if err != nil {
			return err
		}
		bookmarkDir, err := profile.AbsolutePath()
		if err != nil {
			return err
		}

		ch.BkDir = bookmarkDir
		return ch.init(ctx)
	}

	ch.ChromeConfig = NewChromeConfig()
	ch.Profile = p.ID

	if bookmarkDir, err := p.AbsolutePath(); err != nil {
		return err
	} else {
		ch.BkDir = bookmarkDir
	}

	return ch.init(ctx)
}

// Init() is the first method called after a browser instance is created
// and registered.
// Return ok, error
func (ch *Chrome) init(_ *modules.Context) error {
	log.Infof("initializing <%s>", ch.Name)
	return ch.setupWatchers()
}

func (ch *Chrome) setupWatchers() error {
	bookmarkDir := ch.BkDir
	log.Debugf("Watching path: %s", bookmarkDir)
	bookmarkPath := filepath.Join(bookmarkDir, ch.BkFile)
	// Setup watcher
	w := &watch.Watch{
		Path:       bookmarkDir,
		EventTypes: []fsnotify.Op{fsnotify.Create},
		EventNames: []string{bookmarkPath},

		// NOTE: it used to be that chrome watcher would go stale after the
		// first event, this is because the bookmark file is created after the
		// browser is started, so we need to reset the watcher after the first
		// event. This is not needed anymore as we are using the watcher to
		// watch the bookmark dir and not the file itself.
		ResetWatch: false,
	}

	ok, err := modules.SetupWatchers(ch.BrowserConfig, w)
	if err != nil {
		log.Error(err)
		return modules.ErrWatcherSetup
	}
	if !ok {
		return modules.ErrWatcherSetup
	}

	return nil
}

func (ch *Chrome) ResetWatcher() error {
	w := ch.GetWatcher()
	if err := w.W.Close(); err != nil {
		return err
	}
	if err := ch.setupWatchers(); err != nil {
		return err
	}

	go watch.WatchLoop(ch)
	return nil
}

// Returns a pointer to an initialized browser config
func (ch Chrome) Config() *modules.BrowserConfig {
	return ch.BrowserConfig
}

func (ch Chrome) ModInfo() modules.ModInfo {
	return modules.ModInfo{
		ID: modules.ModID(ch.Name),
		New: func() modules.Module {
			return NewChrome()
		},
	}
}

func (ch *Chrome) Watch() *watch.WatchDescriptor {
	// calls modules.BrowserConfig.GetWatcher()
	return ch.GetWatcher()
}

func (ch *Chrome) Run() {
	ch.run(true)
}

func (ch *Chrome) run(runTask bool) {
	startRun := time.Now()

	// Rebuild node tree
	ch.NodeTree = &tree.Node{
		Title:  RootNodeName,
		Parent: nil,
		Type:   tree.RootNode,
	}

	// Load bookmark file
	bookmarkPath, err := ch.BookmarkPath()
	if err != nil {
		log.Error(err)
		return
	}

	var parseChildren ParseChildJSONFunc
	var jsonParseRecursive RecursiveParseJSONFunc

	parseChildren = func(childVal []byte,
		dataType jsonparser.ValueType,
		offset int,
		err error) {
		if err != nil {
			log.Fatal(err)
		}

		err = jsonParseRecursive(nil, childVal, dataType, offset)
		if err != nil {
			log.Error(err)
		}

	}

	// Needed to store the parent of each child node
	var parentNodes []*tree.Node

	jsonParseRoots := func(key []byte,
		node []byte,
		dataType jsonparser.ValueType,
		offset int,
	) error {

		// If node type is string ignore (needed for sync_transaction_version)
		if dataType == jsonparser.String {
			return nil
		}

		ch.IncNodeCount()
		rawNode := new(RawNode)
		rawNode.parseItems(node)

		//log.Debugf("Parsing root folder %s", rawNode.name)

		currentNode := rawNode.getNode(ch)

		// Process this node as parent node later
		parentNodes = append(parentNodes, currentNode)

		// add the root node as parent to this node
		currentNode.Parent = ch.NodeTree

		// Add this root node as a child of the root node
		ch.NodeTree.Children = append(ch.NodeTree.Children, currentNode)

		// Call recursive parsing of this node which must
		// a root folder node
		jsonparser.ArrayEach(node, parseChildren, jsonNodePaths.Children)

		// Finished parsing this root, it is not anymore a parent
		_, parentNodes = parentNodes[len(parentNodes)-1],
			parentNodes[:len(parentNodes)-1]

		//log.Debugf("Parsed root %s folder", rawNode.name)

		return nil
	}

	// Main recursive parsing underneath each root folder
	jsonParseRecursive = func(_ []byte, //key
		node []byte,
		dataType jsonparser.ValueType,
		_ int, //offset
	) error {

		// If node type is string ignore (needed for sync_transaction_version)
		if dataType == jsonparser.String {
			return nil
		}

		ch.IncNodeCount()

		rawNode := new(RawNode)
		rawNode.parseItems(node)

		currentNode := rawNode.getNode(ch)
		//log.Debugf("parsing node %s", currentNode.Name)

		// if parents array is not empty
		if len(parentNodes) != 0 {
			parent := parentNodes[len(parentNodes)-1]
			//log.Debugf("Adding current node to parent %s", parent.Name)

			// Add current node to closest parent
			currentNode.Parent = parent

			// Add current node as child to parent
			currentNode.Parent.Children = append(currentNode.Parent.Children, currentNode)
		}

		// if node is a folder with children
		if rawNode.childrenType == jsonparser.Array && len(rawNode.children) > 2 { // if len(children) > len("[]")

			//log.Debugf("Started folder %s", rawNode.name)
			parentNodes = append(parentNodes, currentNode)

			// Recursively process all child nodes of this folder node
			jsonparser.ArrayEach(node, parseChildren, jsonNodePaths.Children)

			//log.Debugf("Finished folder %s", rawNode.name)
			_, parentNodes = parentNodes[len(parentNodes)-1], parentNodes[:len(parentNodes)-1]

		}

		// if node is url(leaf), handle the url
		if currentNode.Type == tree.URLNode {
			currentNode.URL = string(rawNode.url)
			ch.IncURLCount()

			progress := ch.Progress()
			if progress-ch.lastSentProgress >= 0.05 || progress == 1 {
				ch.lastSentProgress = progress
				go func() {
					msg := events.ProgressUpdateMsg{
						ID:           ch.ModInfo().ID,
						Instance:     ch,
						CurrentCount: ch.URLCount(),
						Total:        ch.Total(),
					}
					if runTask {
						msg.NewBk = true
					}
					events.TUIBus <- msg
				}()
			}

			// Check if url-node already in index
			var nodeVal *tree.Node
			iVal, found := ch.URLIndex.Get(currentNode.URL)

			nameHash := xxhash.ChecksumString64(currentNode.Title)

			// If node url not in index, add it to index
			if !found {
				//log.Debugf("Not found")

				// store hash(name)
				currentNode.NameHash = nameHash

				// The value in the index will be a
				// pointer to currentNode
				log.Tracef("Inserting url %s to index", currentNode.URL)
				ch.URLIndex.Insert(currentNode.URL, currentNode)

				// Run registered bookmark parsing hooks
				err = ch.CallHooks(currentNode)
				if err != nil {
					return err
				}

				// If we find the node already in index
				// we check if the hash(name) changed  meaning
				// the data changed
			} else {
				// log.Debugf("URL Found in index")
				nodeVal = iVal.(*tree.Node)

				// hash(name) is different meaning new commands/tags could
				// be added, we need to process the parsing hoos
				if nodeVal.NameHash != nameHash {
					// log.Debugf("URL name changed !")

					// Run parse hooks on node
					ch.CallHooks(currentNode)
				}

				// Else we do nothing, the node will not
				// change
			}

			//If parent is folder, add it as tag and add current node as child
			//And add this link as child
			if currentNode.Parent.Type == tree.FolderNode {
				// log.Debug("Parent is folder, parsing as tag ...")
				currentNode.Tags = append(currentNode.Tags, currentNode.Parent.Title)
			}
		}

		return nil
	}

	f, err := os.ReadFile(bookmarkPath)
	if err != nil {
		log.Error(err)
		return
	}

	// starts from the "roots" key of chrome json bookmark file
	rootsData, _, _, _ := jsonparser.Get(f, "roots")

	// Start a new node tree building job
	jsonparser.ObjectEach(rootsData, jsonParseRoots)
	ch.SetLastTreeParseRuntime(time.Since(startRun))
	log.Debugf("<%s> parsed tree in %s", ch.Name, ch.LastFullTreeParseRT())
	// Finished node tree building job

	// Debug walk tree
	//go PrintTree(ch.NodeTree)

	// Reset the index to represent the nodetree
	ch.RebuildIndex()

	// Finished parsing
	log.Debugf("<%s> parsed %d bookmarks and %d nodes", ch.Name, ch.URLCount(), ch.NodeCount())

	//Add nodeTree to Cache
	//log.Debugf("<%s> buffer content", ch.Name)
	//ch.BufferDB.Print()

	log.Debugf("<%s> syncing to buffer", ch.Name)
	database.SyncTreeToBuffer(ch.NodeTree, ch.BufferDB)
	log.Debugf("<%s> tree synced to buffer", ch.Name)

	//ch.BufferDB.Print()

	// database.Cache represents bookmarks across all browsers
	// From browsers it should support: add/update
	// Delete method should only be possible through admin interface
	//NOTE: We could have an @ignore command to ignore a bookmark

	// URLIndex is a hashmap index of all URLS representing current state
	// of the browser

	// nodeTree is current state of the browser as tree

	// Buffer is the current state of the browser represetned by
	// URLIndex and nodeTree

	// debugPrint("%d", BufferDB.Count())

	// If the cache is empty just copy buffer to cache
	// until local db is already populated and preloaded
	if err = ch.BufferDB.SyncToCache(); err != nil {
		log.Errorf("syncing buffer to cache: %v", err)
	}

	database.ScheduleBackupToDisk()
	ch.SetLastWatchRuntime(time.Since(startRun))
}

// PreLoad() will be called right after a browser is initialized
func (ch *Chrome) PreLoad(_ *modules.Context) error {
	bookmarkPath, err := ch.BookmarkPath()
	if err != nil {
		log.Error(err)
	}
	ch.SetTotal(preCountCountUrls(bookmarkPath))

	// Send total to msg bus
	go func() {
		events.TUIBus <- events.StartedLoadingMsg{
			ID:    modules.ModID(ch.Name),
			Total: ch.Total(),
		}
	}()

	go ch.run(false)
	return nil
}

// Implement modules.Shutdowner
func (ch *Chrome) Shutdown() error {
	return nil
}

func NewChrome() *Chrome {
	return &Chrome{
		ChromeConfig: ChromeCfg,
		Counter:      &parsing.BrowserCounter{},
	}
}

func init() {
	modules.RegisterBrowser(Chrome{ChromeConfig: ChromeCfg})
}

// interface guards

var _ modules.BrowserModule = (*Chrome)(nil)
var _ modules.ProfileInitializer = (*Chrome)(nil)
var _ modules.PreLoader = (*Chrome)(nil)
var _ watch.WatchRunner = (*Chrome)(nil)
var _ hooks.HookRunner = (*Chrome)(nil)
var _ parsing.Counter = (*Chrome)(nil)
var _ profiles.ProfileManager = (*Chrome)(nil)
