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
//
// Watch package provides functionality for watching file system events and
// managing bookmark data sources.
package watch

import (
	"fmt"
	"slices"
	"time"

	"github.com/blob42/gosuki"
	"github.com/blob42/gosuki/internal/database"
	"github.com/blob42/gosuki/pkg/logging"
	"github.com/blob42/gosuki/pkg/manager"
	"github.com/blob42/gosuki/pkg/parsing"

	"github.com/fsnotify/fsnotify"
)

var (
	log = logging.GetLogger("WATCH")
)

type EventType int

// Modules the implement their bookmark loading through a Run() method with an
// internal logic of handling bookmarks and direct sync with gosuki DB
// Mostly used through implementing [WatchRunner]
type Runner interface {
	Run()
}

type WatchRunner interface {
	Watcher
	Runner
}

// Loader is an interface for modules that can load bookmarks. It requires the implementation of a Load method,
// which returns a slice of pointers to gosuki.Bookmark and an error.
type Loader interface {
	Load() ([]*gosuki.Bookmark, error)
}

// WatchLoader is an interface that combines the capabilities of both Watcher and Loader interfaces.
// It is intended for modules that need to watch for changes and also load bookmarks through the Load method.
type WatchLoader interface {
	Watcher
	Loader

	Name() string // module name
}

type Poller interface {
	Fetcher
	Interval() time.Duration
}

// Fetcher is an interface for modules that fetches data from some source
// and produces a list of bookmarks.
type Fetcher interface {
	Fetch() ([]*gosuki.Bookmark, error)
}

// If the browser needs the watcher to be reset for each new event
type ResetWatcher interface {
	ResetWatcher() error // resets a new watcher
}

// Required interface to be implemented by browsers that want to use the
// fsnotify event loop and watch changes on bookmark files.
type Watcher interface {
	Watch() *WatchDescriptor
}

type Shutdowner interface {
	Shutdown() error
}

// StatMaker interface can be implemented in modules that keep and track stats
type StatMaker interface {
	ResetStats()
}

// WatchDescriptor is a warpper around an fsnotify.Watcher that defines watch properties.
type WatchDescriptor struct {
	// ID is a unique identifier for the watch descriptor.
	ID string

	// W is the underlying fsnotify.Watcher that this wrapper uses to monitor file system events.
	W *fsnotify.Watcher

	// Watches is a slice of pointers to Watch objects, which represent specific files or directories being watched.
	Watches []*Watch

	// eventsChan is a channel used for communicating events related to the watches. It's buffered and has a size determined by fsnotify.BufferSize().
	eventsChan chan fsnotify.Event

	// isWatching is a boolean flag that indicates whether this WatchDescriptor is actively watching any file or directory.
	isWatching bool

	// List of unique event names that where encountered
	// Useful to track unique filenames in a watched path
	TrackEventNames bool
	EventNames      []string
}

func (w *WatchDescriptor) AddEventName(name string) {
	if !slices.Contains(w.EventNames, name) {
		w.EventNames = append(w.EventNames, name)
	}
}

func (w WatchDescriptor) hasReducer() bool {
	return w.eventsChan != nil
}

func NewWatcherWithReducer(name string, reducerLen int, watches ...*Watch) (*WatchDescriptor, error) {
	w, err := NewWatcher(name, watches...)
	if err != nil {
		return nil, err
	}
	w.eventsChan = make(chan fsnotify.Event, reducerLen)

	return w, nil
}

func NewWatcher(name string, watches ...*Watch) (*WatchDescriptor, error) {

	fswatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating fsnotify watcher: %w", err)
	}

	watchedMap := make(map[string]*Watch)
	for _, v := range watches {
		watchedMap[v.Path] = v
	}

	watcher := &WatchDescriptor{
		ID:         name,
		W:          fswatcher,
		Watches:    watches,
		eventsChan: nil,
	}

	// Add all watched paths
	for _, v := range watches {

		err = watcher.W.Add(v.Path)
		if err != nil {
			return nil, fmt.Errorf("adding watch path: %s", v.Path)
		}
	}
	return watcher, nil
}

// Watch is a filesystem object that can be watched for changes.
type Watch struct {
	Path       string        // Path to watch for events
	EventTypes []fsnotify.Op // events to watch for

	// event names to watch for (file/dir names)
	// use "*" to watch all names
	EventNames []string

	// Reset the watcher at each event occurence (useful for `create` events)
	ResetWatch bool
}

// Implement work unit for watchers
type WatchWork struct {
	WatchRunner
}

func (w WatchWork) Run(m manager.UnitManager) {
	watchRun(w, m)
}

func WatchWorkers(m *manager.Manager) []*WatchWork {
	res := []*WatchWork{}
	workers := m.Units()
	for _, w := range workers {
		ww, ok := w.(WatchWork)
		if ok {
			res = append(res, &ww)
		}
	}
	return res
}

// A module that needs to watch paths to load boomkarks. Browsers SHOULD use
// WatchRunner instead.
type WatchLoad struct {
	WatchLoader
}

// Implement work unit for watch loaders
func (w WatchLoad) Run(m manager.UnitManager) {
	watchRun(w, m)
}

// shared watch running implementation for all units that rely on FS watching
func watchRun(w Watcher, m manager.UnitManager) {
	watcher := w.Watch()
	if !watcher.isWatching {
		switch w := w.(type) {
		case WatchLoad:
			go WatchLoop(w.WatchLoader)
		case WatchWork:
			go WatchLoop(w.WatchRunner)
		}

		watcher.isWatching = true

		for _, watch := range watcher.Watches {
			log.Debugf("Watching %s", watch.Path)
		}
	}

	// wait for stop signal
	<-m.ShouldStop()

	// if module implements shutdowner
	sht, ok := w.(Shutdowner)
	if ok {
		if err := sht.Shutdown(); err != nil {
			m.Panic(err)
		}
	}
	m.Done()
}

// Implement work unit for poller runners
type PollWork struct {
	Name string

	Poller
}

func (iw PollWork) Run(m manager.UnitManager) {
	go Poll(iw.Poller, iw.Name)
	// wait for stop signal
	<-m.ShouldStop()
	m.Done()
}

// Main gorouting for polling bookmarks at regular intervals
// One goroutine spawned per module
func Poll(ir Poller, modName string) {

	log.Debug("polling", "module", modName, "interval", ir.Interval())
	beat := time.NewTicker(ir.Interval()).C

	if err := database.LoadBookmarks(ir.Fetch, modName); err != nil {
		log.Error("loading bookmarks", "module", modName, "err", err)
	}
	for range beat {
		if err := database.LoadBookmarks(ir.Fetch, modName); err != nil {
			log.Error("loading bookmarks", "module", modName, "err", err)
		}
	}

}

// Main thread for watching file changes
func WatchLoop(w any) {
	var watcher Watcher

	watcher, ok := w.(Watcher)
	if !ok {
		log.Errorf("%v does not implement Watcher", w)
	}

	watch := watcher.Watch()
	beat := time.NewTicker(1 * time.Second).C
	log.Debugf("<%s> Started watcher", watch.ID)
watchloop:
	for {

		select {
		case <-beat:
		// log.Debugf("main watch loop beat %s", watcher.ID)
		case event := <-watch.W.Events:
			// Very verbose
			// log.Debug("event", "OP", event.Op, "eventName", event.Name)

			// On Chrome like browsers the bookmarks file is created
			// at every change.

			/*
			* When a file inside a watched directory is renamed/created,
			* fsnotify does not seem to resume watching the newly created file, we
			* need to destroy and create a new watcher. The ResetWatcher() and
			* `break` statement ensure we get out of the `select` block and catch
			* the newly created watcher to catch events even after rename/create op
			*
			* NOTE: this does not seem to be an issue anymore.
			* Leaving comment until further testing
			 */

			for _, watched := range watch.Watches {
				if watch.TrackEventNames {
					watch.AddEventName(event.Name)
				}
				for _, watchedEv := range watched.EventTypes {
					for _, watchedName := range watched.EventNames {
						log.Debug("event", "OP", event.Op, "eventName", event.Name)

						if event.Op&watchedEv == watchedEv &&
							(watchedName == "*" || event.Name == watchedName) {

							// For watchers who use a reducer forward the event
							// to the reducer channel
							if watch.hasReducer() {
								ch := watch.eventsChan
								ch <- event

								// the reducer will call Run()
							} else {
								go func() {
									if counter, ok := w.(parsing.Counter); ok {
										counter.ResetCount()
									}
									if runner, ok := w.(Runner); ok {
										runner.Run()
									} else if loader, ok := w.(WatchLoader); ok {
										if err := database.LoadBookmarks(loader.Load, loader.Name()); err != nil {
											log.Errorf("loading bookmarks: %s", err)
										}
									}
								}()
							}

							// log.Debug("event", event.Op, "eventName", event.Name)
							if watched.ResetWatch {
								log.Debugf("resetting watchers")
								if r, ok := w.(ResetWatcher); ok {
									r.ResetWatcher()
									// break out of watch loop
									break watchloop
								} else {
									log.Fatalf("<%s> does not implement ResetWatcher", watch.ID)
								}
							}

						}
					}
				}
			}

		case err := <-watch.W.Errors:
			if err != nil {
				log.Error(err)
			}
		}
	}
}
