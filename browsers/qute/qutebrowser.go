//
//  Copyright (c) 2024-2025 Chakib Ben Ziane <contact@blob42.xyz>  and [`gosuki` contributors](https://github.com/blob42/gosuki/graphs/contributors).
//  All rights reserved.
//
//  SPDX-License-Identifier: AGPL-3.0-or-later
//
//  This file is part of GoSuki.
//
//  GoSuki is free software: you can redistribute it and/or modify
//  it under the terms of the GNU Affero General Public License as
//  published by the Free Software Foundation, either version 3 of the
//  License, or (at your option) any later version.
//
//  GoSuki is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of
//  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU Affero General Public License for more details.
//
//  You should have received a copy of the GNU Affero General Public License
//  along with gosuki.  If not, see <http://www.gnu.org/licenses/>.
//

package qute

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/blob42/gosuki"
	"github.com/blob42/gosuki/hooks"
	"github.com/blob42/gosuki/internal/database"
	"github.com/blob42/gosuki/internal/utils"
	"github.com/blob42/gosuki/pkg/events"
	"github.com/blob42/gosuki/pkg/modules"
	"github.com/blob42/gosuki/pkg/parsing"
	"github.com/blob42/gosuki/pkg/watch"
)

// Qute browser module
type Qute struct {
	// holds browsers.BrowserConfig
	*QuteConfig
	parsing.Counter
	lastSentProgress float64
}

// PreCount implements parsing.Counter.
func (qu *Qute) preCountUrls() error {
	var count int
	var err error

	bkPath, err := qu.BookmarkPath()
	if err != nil {
		return fmt.Errorf("%s : %w", bkPath, err)
	}

	bkFile, err := os.Open(bkPath)
	if err != nil {
		return fmt.Errorf("open %s : %w", bkPath, err)
	}
	defer bkFile.Close()
	if count, err = utils.CountLines(bkFile); err != nil {
		return fmt.Errorf("reading file %s : %w", bkPath, err)
	}
	qu.AddTotal(uint(count))

	qmFile, err := os.Open(qu.quickmarksPath)
	if err != nil {
		return fmt.Errorf("open %s : %w", qu.quickmarksPath, err)
	}

	if count, err = utils.CountLines(qmFile); err != nil {
		return fmt.Errorf("reading file %s : %w", qu.quickmarksPath, err)
	}

	qu.AddTotal(uint(count))

	// Send total to msg bus
	go func() {
		events.TUIBus <- events.StartedLoadingMsg{
			ID:    modules.ModID(qu.Name),
			Total: qu.Total(),
		}
	}()

	return nil
}

// Detect implements modules.Detector.
func (qu *Qute) Detect() ([]modules.Detected, error) {
	res := []modules.Detected{}
	bPath, err := utils.ExpandOnly(qu.BaseDir)
	if err != nil {
		return res, err
	}
	exist, err := utils.DirExists(bPath)
	if err != nil {
		return res, err
	}

	if exist {
		res = append(res, modules.Detected{
			Flavour:  BrowserName,
			BasePath: bPath,
		})
	}

	return res, nil
}

func (qu Qute) Init(ctx *modules.Context) error {
	var err error

	if ctx.IsTUI {
		QuteCfg.TUI = true
	}

	// This section handles symlinks to qutebrowser
	// Typically the case with dotfiles.
	qu.quickmarksPath, err = utils.ExpandOnly(qu.quickmarksPath)
	if err != nil {
		return err
	}

	isSym, err := utils.IsSymlink(qu.quickmarksPath)
	if err != nil {
		return err
	}

	// set parent directory as the new base dir
	if isSym {
		fullQuickmarksPath, err := filepath.EvalSymlinks(qu.quickmarksPath)
		if err != nil {
			return err
		}

		qu.quickmarksPath = fullQuickmarksPath
		qu.BkDir = filepath.Dir(fullQuickmarksPath) + "/bookmarks"
		qu.BaseDir = filepath.Dir(fullQuickmarksPath)
	} else {
		qu.BaseDir, err = utils.ExpandPath(qu.BaseDir)
		if err != nil {
			return fmt.Errorf("expadning %s : %w", qu.BaseDir, err)
		}

	}

	return qu.setupWatchers()
}

func (qu Qute) setupWatchers() error {
	bookmarkPath, err := qu.BookmarkPath()
	if err != nil {
		return fmt.Errorf("%s : %w", bookmarkPath, err)
	}

	bookmarkDir, err := utils.ExpandPath(qu.BkDir)
	if err != nil {
		return fmt.Errorf("expanding %s : %w", qu.BkDir, err)
	}

	w := &watch.Watch{
		Path:       bookmarkDir,
		EventTypes: []fsnotify.Op{fsnotify.Create},
		EventNames: []string{bookmarkPath},
	}

	wQuickmarks := &watch.Watch{
		Path:       qu.BaseDir,
		EventTypes: []fsnotify.Op{fsnotify.Create, fsnotify.Write},
		EventNames: []string{qu.quickmarksPath},
	}

	ok, err := modules.SetupWatchers(qu.BrowserConfig, w, wQuickmarks)
	if err != nil {
		return fmt.Errorf("could not setup watcher: %w", err)
	}
	if !ok {
		return errors.New("could not setup watcher")
	}

	return nil
}

func (qu Qute) Config() *modules.BrowserConfig {
	return qu.BrowserConfig
}

func (qu Qute) ModInfo() modules.ModInfo {
	return modules.ModInfo{
		ID: modules.ModID(qu.Name),
		New: func() modules.Module {
			return NewQute()
		},
	}
}

func (qu *Qute) Run() {
	err := qu.load(true)
	if err != nil {
		log.Error(err)
	}
}

func (qu *Qute) loadBookmarks(runTask bool) error {
	var err error

	bkPath, err := qu.BookmarkPath()
	if err != nil {
		return err
	}

	bkFile, err := os.Open(bkPath)
	if err != nil {
		return err
	}
	defer bkFile.Close()

	reader := bufio.NewReader(bkFile)
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		} else if err == io.EOF {
			break
		}

		fields := strings.Fields(line)

		bk := &gosuki.Bookmark{
			URL: strings.TrimSpace(fields[0]),
			Title: strings.TrimSpace(
				strings.Join(fields[1:], " "),
			),
			Desc:   "",
			Module: qu.Name,
		}

		qu.CallHooks(bk)

		qu.BufferDB.UpsertBookmark(bk)
		qu.IncURLCount()
		qu.trackProgress(runTask)
	}

	return nil
}

func (qu *Qute) loadQuickMarks(runTask bool) error {
	qmFile, err := os.Open(qu.quickmarksPath)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(qmFile)

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read line: %v", err)
		} else if err == io.EOF {
			break
		}

		qu.IncURLCount()
		if runTask {
			qu.AddTotal(1)
		}
		qu.trackProgress(runTask)

		fields := strings.Fields(line)

		bk := &gosuki.Bookmark{
			URL:    strings.TrimSpace(fields[len(fields)-1]), // Last field is the URL
			Tags:   fields[:len(fields)-1],
			Module: qu.Name,
		}

		// Call hooks on bookmark instead of node
		err = qu.CallHooks(bk)
		if err != nil {
			return err
		}

		err = qu.BufferDB.UpsertBookmark(bk)
		if err != nil {
			log.Errorf("db upsert: %s", bk.URL)
		}
	}

	return nil
}

func (qu *Qute) trackProgress(runTask bool) {
	progress := qu.Progress()
	if progress-qu.lastSentProgress >= 0.05 || progress == 1 {
		qu.lastSentProgress = progress
		if QuteCfg.TUI {
			go func() {
				msg := events.ProgressUpdateMsg{
					ID:           qu.ModInfo().ID,
					Instance:     qu,
					CurrentCount: qu.URLCount(),
					Total:        qu.Total(),
				}
				if runTask {
					msg.NewBk = true
				}
				events.TUIBus <- msg
			}()
		}
	}
}

func (qu *Qute) load(runTask bool) error {
	// Initial Loading, precounting urls
	if !runTask {
		if err := qu.preCountUrls(); err != nil {
			return err
		}
	}

	// Loading logic
	startWork := time.Now()
	err := qu.loadBookmarks(runTask)
	if err != nil {
		return err
	}

	err = qu.loadQuickMarks(runTask)
	if err != nil {
		return err
	}

	qu.SetLastTreeParseRuntime(time.Since(startWork))
	log.Debugf("<%s> loaded bookmarks in %s", qu.Name, qu.LastFullTreeParseRT())

	err = qu.BufferDB.SyncToCache()
	if err != nil {
		log.Errorf("<%s>: %v", qu.Name, err)
	}

	database.ScheduleBackupToDisk()
	qu.SetLastWatchRuntime(time.Since(startWork))

	return err
}

func (qu *Qute) PreLoad(_ *modules.Context) error {
	return qu.load(false)
}

func (qu *Qute) Watch() *watch.WatchDescriptor {
	// calls modules.BrowserConfig.GetWatcher()
	return qu.GetWatcher()
}

// Implement modules.Shutdowner
func (qu *Qute) Shutdown() error {
	return nil
}

func NewQute() *Qute {
	return &Qute{
		QuteConfig: QuteCfg,
		Counter:    &parsing.BrowserCounter{},
	}
}

func init() {
	modules.RegisterBrowser(Qute{QuteConfig: QuteCfg})
}

// interface guards

var _ modules.BrowserModule = (*Qute)(nil)
var _ modules.Initializer = (*Qute)(nil)

var _ modules.Detector = (*Qute)(nil)
var _ watch.WatchRunner = (*Qute)(nil)
var _ modules.PreLoader = (*Qute)(nil)
var _ parsing.Counter = (*Qute)(nil)
var _ hooks.HookRunner = (*Qute)(nil)
