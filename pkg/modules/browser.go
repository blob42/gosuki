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

package modules

import (
	"fmt"
	"time"

	"github.com/blob42/gosuki/hooks"
	"github.com/blob42/gosuki/internal/database"
	"github.com/blob42/gosuki/internal/index"
	"github.com/blob42/gosuki/internal/logging"
	"github.com/blob42/gosuki/internal/utils"
	"github.com/blob42/gosuki/pkg/parsing"
	"github.com/blob42/gosuki/pkg/profiles"
	"github.com/blob42/gosuki/pkg/tree"
	"github.com/blob42/gosuki/pkg/watch"
)

var registeredBrowsers []BrowserModule

type BrowserType uint8

// Browser types
const (
	// Chromium based browsers (chrome, brave ... )
	TChrome BrowserType = iota

	// Firefox based browsers ie. they relay on places.sqlite
	TFirefox

	// Other
	TCustom
)

// reducer channel length, bigger means less sensitivity to events
var (
	log            = logging.GetLogger("BASE")
	ReducerChanLen = 1000
)

type Browser interface {
	// Returns a pointer to an initialized browser config
	Config() *BrowserConfig
}

// The profile preferences for modules with builtin profile management.
type ProfilePrefs struct {

	// Whether to watch all the profiles for multi-profile modules
	WatchAllProfiles bool `toml:"watch_all_profiles" mapstructure:"watch_all_profiles"`
	Profile          string `toml:"profile" mapstructure:"profile"`
}

// BrowserConfig is the main browser configuration shared by all browser modules.
type BrowserConfig struct {
	Name string
	Type BrowserType

	// Absolute path to the browser's bookmark directory
	BkDir string

	// Name of bookmarks file
	BkFile string

	// In memory sqlite db (named `memcache`).
	// Used to keep a browser's state of bookmarks across jobs.
	BufferDB *database.DB

	// Fast query db using an RB-Tree hashmap.
	// It represents a URL index of the last running job
	URLIndex index.HashTree

	// Pointer to the root of the node tree
	// The node tree is built again for every Run job on a browser
	NodeTree *tree.Node
	// Various parsing and timing stats
	*parsing.Stats

	watcher        *watch.WatchDescriptor
	UseFileWatcher bool

	// Hooks registered by the browser module identified by name
	UseHooks   []string

	// Registered hooks
	hooks map[string]hooks.Hook
}


func (b *BrowserConfig) GetWatcher() *watch.WatchDescriptor {
	return b.watcher
}

// CallHooks calls all registered hooks for this browser for the given
// *tree.Node. The hooks are called in the order they were registered. This is
// usually done within the parsing logic of a browser module, typically in the
// Run() method.
func (b BrowserConfig) CallHooks(node *tree.Node) error {

	if node == nil {
		return fmt.Errorf("hook node is nil")
	}

	for _, hook := range b.hooks {
		log.Debugf("<%s> calling hook <%s> on node <%s>",b.Name, hook.Name, node.URL)
		if err := hook.Func(node); err != nil {
			return err
		}
	}
	return nil
}

// Registers hooks for this browser. Hooks are identified by their name.
func (b BrowserConfig) AddHooks(hooks ...hooks.Hook) {
 	for _, hook := range hooks {
		b.hooks[hook.Name] = hook
	}
}


func (b BrowserConfig) HasHook(hook hooks.Hook) bool {
	_, ok := b.hooks[hook.Name]
	return ok
}


//TODO!: use this method instead of manually building bookmark path
// BookmarkPath returns the absolute path to the bookmark file.
// It expands the path by concatenating the base directory and bookmarks file, 
// then checks if it exists.
func (b BrowserConfig) BookmarkPath() (string, error) {
	bPath, err := utils.ExpandPath(b.BkDir, b.BkFile)
	if err != nil {
		log.Error(err)
	}

	exists, err := utils.CheckFileExists(bPath)
	if err != nil {
		return "", err
	}

	if !exists {
		return "", fmt.Errorf("not a bookmark path: %s ", bPath)
	}

	return bPath, nil
}

// Rebuilds the memory url index after parsing all bookmarks.
// Keeps the memory url index in sync with last known state of browser bookmarks
func (b BrowserConfig) RebuildIndex() {
	start := time.Now()
	log.Debugf("<%s> rebuilding index based on current nodeTree", b.Name)
	b.URLIndex = index.NewIndex()
	tree.WalkBuildIndex(b.NodeTree, b.URLIndex)
	log.Debugf("<%s> index rebuilt in %s", b.Name, time.Since(start))
}

func (b BrowserConfig) ResetStats() {
	log.Debugf("<%s> resetting stats", b.Name)
	b.LastURLCount = b.CurrentURLCount
	b.LastNodeCount = b.CurrentNodeCount
	b.CurrentNodeCount = 0
	b.CurrentURLCount = 0
}


// Browser who implement this interface need to handle all shuttind down and
// cleanup logic in the defined methods. This is usually called at the end of
// the browser instance lifetime
type Shutdowner interface {
	watch.Shutdowner
}

// Loader is an interface for modules which is run only once when the module
// starts. It should have the same effect as  Watchable.Run().
// Run() is automatically called for watched events, Load() is called once
// before starting to watch events. 
//
// Loader allows modules to do a first pass of Run() logic before the watcher
// threads is spawned 
type Loader interface {

	// Load() will be called right after a browser is initialized
	Load() error
}

// Initialize the module before any data loading or callbacks
// If a module wants to do any preparation and prepare custom state before Loader.Load()
// is called and before any Watchable.Run() or other callbacks are executed.
type Initializer interface {

	// Init() is the first method called after a browser instance is created
	// and registered.
	// A pointer to 
	// Return ok, error
	Init(*Context) error
}

// ProfileInitializer is similar to Initializer but is called with a profile.
// This is useful for modules that need to do some custom initialization for a
// specific profile.
type ProfileInitializer interface {
	Init(*Context, *profiles.Profile) error
}

// SetupBrowser() is called for every browser module. It sets up the browser and calls
// the following methods if they are implemented by the module:
//
// 	1. [Initializer].Init() : state initialization
// 	2. [Loader].Load(): Do the first loading of data (ex first loading of bookmarks )
func SetupBrowser(browser BrowserModule, c *Context, p *profiles.Profile) error {



	log.Infof("setting up browser <%s>", browser.ModInfo().ID)
	browserID := browser.ModInfo().ID

	// Handle Initializers custom Init from Browser module
	initializer, okInit := browser.(Initializer)
	pInitializer, okProfileInit := browser.(ProfileInitializer)

	if okProfileInit && p == nil {
		 log.Warningf("<%s> ProfileInitializer called with nil profile", browserID)
	}

	if !okProfileInit  && !okInit {
		log.Warningf("<%s> does not implement Initializer or ProfileInitializer, not calling Init()", browserID)
	}

	if okInit {
		log.Debugf("<%s> custom init", browserID)
		//TODO!: missing profile name
		if err := initializer.Init(c); err != nil {
			return fmt.Errorf("<%s> initialization error: %w", browserID, err)
		}
	} 

	// Handle Initializers custom Init from Browser module
	if okProfileInit {
		if p != nil {
			log.Debugf("<%s> custom init with profile <%s>", browserID, p.Name)
		}

		if err := pInitializer.Init(c, p); err != nil {
			return fmt.Errorf("<%s> initialization error: %w", browserID, err)
		}
	}

	// We modify the base config after the custom init had the chance to
	// modify it (ex. set the profile name)

    bConf := browser.Config()

	// Setup registered hooks
	bConf.hooks = make(map[string]hooks.Hook)
	for _, hookName := range bConf.UseHooks {
		hook, ok := hooks.Predefined[hookName]
		if !ok {
			return fmt.Errorf("hook <%s> not defined", hookName)
		}
		bConf.AddHooks(hook)
	}


	// Init browsers' BufferDB
	buffer, err := database.NewBuffer(bConf.Name)
	if err != nil {
		return err
	}
	bConf.BufferDB = buffer

	// Creates in memory Index (RB-Tree)
	bConf.URLIndex = index.NewIndex()


	// Default browser loading logic
	// Make sure that cache is initialized
	if !database.Cache.IsInitialized() {
		return fmt.Errorf("<%s> Loading bookmarks while cache not yet initialized", browserID)
	}

	// handle Loader interface
	loader, ok := browser.(Loader)
	if ok {
		log.Debugf("<%s> custom loading", browserID)
		err := loader.Load()
		if err != nil {
			return fmt.Errorf("loading error <%s>: %v", browserID, err)
			// continue
		}
	}
	return nil
}

// Sets up a watcher service using the provided []Watch elements
// Returns true if a new watcher was created. false if it was previously craeted
// or if the browser does not need a watcher (UseFileWatcher == false).
func SetupWatchers(browserConf *BrowserConfig, watches ...*watch.Watch) (bool, error) {
	var err error
	if !browserConf.UseFileWatcher {
		log.Warningf("<%s> does not use file watcher but asked for it", browserConf.Name)
		return false, nil
	}

	var bkPath string
	if bkPath, err = browserConf.BookmarkPath(); err != nil {
		return false, err
	}

	browserConf.watcher, err = watch.NewWatcher(bkPath, watches...)
	if err != nil {
		return false, err
	}

	return true, nil
}

func SetupWatchersWithReducer(browserConf *BrowserConfig,
	reducerChanLen int,
	watches ...*watch.Watch) (bool, error) {
	var err error

	if !browserConf.UseFileWatcher {
		return false, nil
	}

	var bkPath string
	if bkPath, err = browserConf.BookmarkPath(); err != nil {
		return false, err
	}
	browserConf.watcher, err = watch.NewWatcherWithReducer(bkPath, reducerChanLen, watches...)
	if err != nil {
		return false, err
	}

	return true, nil

}

func RegisterBrowser(browserMod BrowserModule) {
	if err := verifyModule(browserMod); err != nil {
		panic(err)
	}

	registeredBrowsers = append(registeredBrowsers, browserMod)
	
	// A browser module is also a module
	registeredModules = append(registeredModules, browserMod)
}

func GetBrowserModules() []BrowserModule {
	return registeredBrowsers
}
