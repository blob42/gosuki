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

package modules

import (
	"fmt"
	"io/fs"
	"time"

	"github.com/blob42/gosuki"
	"github.com/blob42/gosuki/hooks"
	"github.com/blob42/gosuki/internal/database"
	"github.com/blob42/gosuki/internal/index"
	"github.com/blob42/gosuki/internal/utils"
	"github.com/blob42/gosuki/pkg/logging"
	"github.com/blob42/gosuki/pkg/profiles"
	"github.com/blob42/gosuki/pkg/tree"
	"github.com/blob42/gosuki/pkg/watch"
)

var registeredBrowsers []BrowserModule

type BrowserType uint8

// reducer channel length, bigger means less sensitivity to events
var (
	log            = logging.GetLogger("mod")
	ReducerChanLen = 1000
)

// browser modules need to implement Browser interface
type BrowserModule interface {
	Browser
	Module
}

type Browser interface {
	// Returns a pointer to an initialized browser config
	Config() *BrowserConfig
}

// Browsers must offer a way to detect if they are installed on the system and
// display path to their base directory. Note that if the browser module already
// implements profiles.ProfileManager, implementing this interface is redundant.
type Detector interface {
	// List of detected browser instances
	Detect() ([]Detected, error)
}

type Detected struct {
	Flavour string

	// Canonical path to the browser config directory
	BasePath string
}

// The profile preferences for modules with builtin profile management.
type ProfilePrefs struct {
	// Whether to watch all the profiles for multi-profile modules
	WatchAllProfiles bool `toml:"watch-all-profiles" mapstructure:"watch-all-profiles"`

	Profile string `toml:"profile" mapstructure:"profile"`
}

// BrowserConfig is the main browser configuration shared by all browser modules.
type BrowserConfig struct {
	Name string

	// Path to the browser base config directory
	BaseDir string

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

	watcher *watch.WatchDescriptor

	// Watch for changes on the bookmark file
	UseFileWatcher bool

	// Hooks registered by the browser module identified by name
	UseHooks []string

	// Registered hooks
	hooks []hooks.NamedHook
}

func (b *BrowserConfig) GetWatcher() *watch.WatchDescriptor {
	return b.watcher
}

// CallHooks calls all registered hooks for this browser for the given
// [*tree.Node] or [*gosuki.Bookmark]. The hooks are called in the order they
// were registered. This is usually done within the parsing logic of a browser
// module, typically in the Run() method. These hooks will be called everytime
// browser bookmarks are parsed.
func (b BrowserConfig) CallHooks(obj any) error {

	switch obj := obj.(type) {
	case *tree.Node:
		node := obj
		if node == nil {
			return fmt.Errorf("hook node is nil")
		}

		for _, hook := range b.hooks {
			if (hook.Kind() & hooks.BrowserHook) == 0 {
				continue
			}
			if hook, ok := hook.(hooks.Hook[*tree.Node]); ok {
				log.Tracef("<%s> calling hook <%s> on node <%s>", b.Name, hook.Name(), node.URL)
				if err := hook.Func(node); err != nil {
					return err
				}
			}
		}

	case *gosuki.Bookmark:
		bk := obj
		for _, hook := range b.hooks {
			if (hook.Kind() & hooks.BrowserHook) == 0 {
				continue
			}
			if hook, ok := hook.(hooks.Hook[*gosuki.Bookmark]); ok {
				log.Tracef("<hook:%s> calling  <%s> on <%s>", b.Name, hook.Name(), bk.URL)
				if err := hook.Func(bk); err != nil {
					return err
				}
			}
		}
	default:
		panic("unknown hook type")
	}

	return nil
}

// Registers hooks for this browser. Hooks are identified by their name.
func (b *BrowserConfig) addHooks(bHooks ...hooks.NamedHook) {
	b.hooks = append(b.hooks, bHooks...)
	hooks.SortByPriority(b.hooks)

}

// BookmarkPath returns the absolute path to the bookmark file.
// It expands the path by concatenating the base directory and bookmarks file,
// then checks if it exists.
func (b BrowserConfig) BookmarkPath() (string, error) {
	bPath, err := utils.ExpandPath(b.BkDir, b.BkFile)
	if err != nil {
		return "", err
	}

	exists, err := utils.CheckFileExists(bPath)
	if err != nil {
		return "", err
	}

	if !exists {
		return "", fmt.Errorf("not a bookmark path: %s/%s", b.BkDir, b.BkFile)
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

// SetupBrowser() is called for every browser module. It sets up the browser and calls
// the following methods if they are implemented by the module:
//
//  1. [Initializer].Init() : state initialization
//  2. [PreLoader].Load(): initial preloading of bookmarks
func SetupBrowser(browser BrowserModule, c *Context, p *profiles.Profile) error {

	browserID := browser.ModInfo().ID
	log.Debug("setting up", "browser", browserID)

	// Handle Initializers custom Init from Browser module
	initializer, okInit := browser.(Initializer)
	pInitializer, okProfileInit := browser.(ProfileInitializer)

	if okProfileInit && p == nil {
		log.Warnf("<%s> ProfileInitializer called with nil profile", browserID)
	}

	if !okProfileInit && !okInit {
		log.Warnf("<%s> does not implement Initializer or ProfileInitializer, skipping Init()", browserID)
	}

	if okInit {
		log.Debugf("<%s> custom init", browserID)
		if err := initializer.Init(c); err != nil {
			if _, pathErr := err.(*fs.PathError); pathErr {
				return &ErrModDisabled{MissingPath, err}
			} else if err == ErrWatcherSetup {
				return &ErrModDisabled{MissingPath, err}
			}
			return fmt.Errorf("init: %w", err)
		}
	}

	// Handle Initializers custom Init from Browser module
	if okProfileInit {
		if p != nil {
			log.Debugf("<%s> custom init with profile <%s>", browserID, p.Name)
		}

		if err := pInitializer.Init(c, p); err != nil {
			if err == ErrWatcherSetup {
				return &ErrModDisabled{MissingPath, err}
			}
			return fmt.Errorf("init: %w", err)
		}
	}

	// We modify the base config after the custom init had the chance to
	// modify it (ex. set the profile name)

	bConf := browser.Config()

	// Setup registered hooks
	bConf.hooks = []hooks.NamedHook{}
	for _, hookName := range bConf.UseHooks {
		hook, ok := hooks.Defined[hookName]
		if !ok {
			return fmt.Errorf("hook <%s> not defined", hookName)
		}
		bConf.addHooks(hook)
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

	// handle PreLoader interface
	loader, ok := browser.(PreLoader)
	if ok {
		log.Debugf("<%s> preloading", browserID)
		err := loader.PreLoad(c)
		if err != nil {
			return fmt.Errorf("preloading error <%s>: %v", browserID, err)
			// continue
		}
	}

	return nil
}

// SetupWatchers sets up a filesystem watch using the provided []Watch elements
// Returns true if a new watcher was created. false if it was previously created
// or if the browser does not need a watcher (UseFileWatcher == false).
func SetupWatchers(browserConf *BrowserConfig, watches ...*watch.Watch) (bool, error) {
	var err error
	if !browserConf.UseFileWatcher {
		log.Warnf("<%s> does not use file watcher but asked for it", browserConf.Name)
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

func RegisterBrowser(bm BrowserModule) {
	if err := verifyModule(bm); err != nil {
		panic(err)
	}

	registeredBrowsers = append(registeredBrowsers, bm)

	// A browser module is also a module
	registeredModules = append(registeredModules, bm)
}

func GetBrowserModules() []BrowserModule {
	var result []BrowserModule
	for _, browser := range registeredBrowsers {
		if !disabledMods[browser.ModInfo().ID] {
			result = append(result, browser)
		}
	}
	return result
}
