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
//
// Watch package provides functionality for watching file system events and
// managing bookmark data sources.
package watch

import (
	"time"

	"github.com/blob42/gosuki"
	"github.com/blob42/gosuki/internal/database"
	"github.com/blob42/gosuki/internal/logging"
	"github.com/blob42/gosuki/pkg/manager"

	"github.com/fsnotify/fsnotify"
)

var log = logging.GetLogger("WATCH")
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

type IntervalFetcher interface {
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

// Stats interface can be implemented in modules that keep and track stats
type Stats interface {
	ResetStats()
}

// WatchDescriptor wraps around an fsnotify.Watcher and provides additional
// functionality for managing file system watches.
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
}

func (w WatchDescriptor) hasReducer() bool {
	//TODO: test the type of eventsChan
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
		return nil, err
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
			return nil, err
		}
	}
	return watcher, nil
}

// Watch is a filesystem object that can be watched for changes.
type Watch struct {
	Path       string        // Path to watch for events
	EventTypes []fsnotify.Op // events to watch for
	EventNames []string      // event names to watch for (file/dir names)

	// Reset the watcher at each event occurence (useful for `create` events)
	ResetWatch bool
}

// Implement work unit for watchers
type WatchWork struct {
	WatchRunner
}

func (w WatchWork) Run(m manager.UnitManager) {
	watcher := w.Watch()
	if ! watcher.isWatching {
		go WatchLoop(w.WatchRunner)
		watcher.isWatching = true

		for _, watch := range watcher.Watches{
			log.Debugf("Watching %s", watch.Path)
		}
	}

	// wait for stop signal
	<-m.ShouldStop()
	sht, ok := w.WatchRunner.(Shutdowner)
	if ok {
		if err := sht.Shutdown(); err != nil {
			m.Panic(err)
		}
	}
	m.Done()
}

// Implement work unit for interval runners
type IntervalWork struct {
	Name string
	IntervalFetcher
}

func (iw IntervalWork) Run(m manager.UnitManager) {
	go IntervalLoop(iw.IntervalFetcher, iw.Name)
	// wait for stop signal
	<-m.ShouldStop()
	m.Done()
}

// Main thread for fetching bookmarks at regular intervals
// One goroutine spawned per module
func IntervalLoop(ir IntervalFetcher, modName string) {
	var err error
	var buffer *database.DB

	// prepare buffer for module
	buffer, err = database.NewBuffer(modName)
	if err != nil {
		log.Criticalf("could not create buffer for <%s>: %s", modName, err)
		return
	}
	defer buffer.Close()

	beat := time.NewTicker(ir.Interval()).C
	for range beat {
		marks, err := ir.Fetch()
		if err != nil {
			log.Errorf("error fetching bookmarks: %s", err)
		}

		if len(marks) == 0 {
			log.Warningf("no bookmarks fetched")
			continue
		}

		for _, mark := range marks {
			log.Debugf("Fetched bookmark: %s", mark.URL)
			buffer.UpsertBookmark(mark)
		}
		buffer.PrintBookmarks()
		err = buffer.SyncToCache()
		if err != nil {
			log.Errorf("error syncing buffer to cache: %s", err)
			continue
		}
		database.ScheduleSyncToDisk()
	}

}

// Main thread for watching file changes
func WatchLoop(w WatchRunner) {

	watcher := w.Watch()
	beat := time.NewTicker(1 * time.Second).C
	log.Debugf("<%s> Started watcher", watcher.ID)
watchloop:
	for {

		select {
		case <-beat:
			// log.Debugf("main watch loop beat %s", watcher.ID)
		case event := <-watcher.W.Events:
			// Very verbose
			// log.Debugf("event: %v | eventName: %v", event.Op, event.Name)

			// On Chrome like browsers the bookmarks file is created
			// at every change.

			/*
			 * When a file inside a watched directory is renamed/created,
			 * fsnotify does not seem to resume watching the newly created file, we
			 * need to destroy and create a new watcher. The ResetWatcher() and
			 * `break` statement ensure we get out of the `select` block and catch
			 * the newly created watcher to catch events even after rename/create
			 * 
			 * NOTE: this does not seem to be an issue anymore. More testing
			 * and user feedback is needed. Leaving this comment here for now.
			 */

			for _, watched := range watcher.Watches {
				for _, watchedEv := range watched.EventTypes {
					for _, watchedName := range watched.EventNames {
						// log.Debugf("event: %v | eventName: %v", event.Op, event.Name)

						if event.Op&watchedEv == watchedEv &&
							event.Name == watchedName {

							// For watchers who use a reducer forward the event
							// to the reducer channel
							if watcher.hasReducer() {
								ch := watcher.eventsChan
								ch <- event

								// the reducer will call Run()
							} else {
								go func(){
									w.Run()
									if stats, ok := w.(Stats); ok {
										stats.ResetStats()
									}
								}()
							}

							//log.Warningf("event: %v | eventName: %v", event.Op, event.Name)

							//TODO!: remove condition and use interface instead
							// if the runner inplmenets reset watcher we call
							// its reset watcher
							if watched.ResetWatch {
								log.Debugf("resetting watchers")
								if r, ok := w.(ResetWatcher); ok {
									r.ResetWatcher()
									// break out of watch loop
									break watchloop
								} else {
									log.Fatalf("<%s> does not implement ResetWatcher", watcher.ID)
								}
							}

						}
					}
				}
			}


			// Firefox keeps the file open and makes changes on it
			// It needs a debouncer
			//if event.Name == bookmarkPath {
			//log.Debugf("event: %v | eventName: %v", event.Op, event.Name)
			////go debounce(1000*time.Millisecond, spammyEventsChannel, w)
			//ch := w.EventsChan()
			//ch <- event
			////w.Run()
			//}
		case err := <-watcher.W.Errors:
			if err != nil {
				log.Error(err)
			}
		}
	}
}
