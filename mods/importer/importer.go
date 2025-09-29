//
//  Copyright (c) 2025 Chakib Ben Ziane <contact@blob42.xyz>  and [`gosuki` contributors](https://github.com/blob42/gosuki/graphs/contributors).
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

package mods

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/fsnotify/fsnotify"

	"github.com/blob42/gosuki"
	"github.com/blob42/gosuki/internal/utils"
	"github.com/blob42/gosuki/pkg/config"
	"github.com/blob42/gosuki/pkg/events"
	"github.com/blob42/gosuki/pkg/logging"
	"github.com/blob42/gosuki/pkg/modules"
	"github.com/blob42/gosuki/pkg/parsing"
	"github.com/blob42/gosuki/pkg/watch"
)

const (
	ImporterID = "html-autoimport"
)

var (
	Config *BookmarksImporterConfig
	log    = logging.GetLogger(ImporterID)
	model  *importerModel
)

type importerModel struct {
	watchedPaths []string
	watcher      *watch.WatchDescriptor
}

type BookmarksImporter struct{}

// implements Initializer.
func (im *BookmarksImporter) Init(ctx *modules.Context) error {
	var err error
	var p string
	watches := []*watch.Watch{}

	model.watchedPaths = Config.Paths
	for _, path := range Config.Paths {
		if p, err = utils.ExpandPath(path); err != nil {
			log.Warn(err, "path", path)
			continue
		} else {
			model.watchedPaths = append(model.watchedPaths, p)
		}

		watches = append(watches, &watch.Watch{
			Path:       p,
			EventTypes: []fsnotify.Op{fsnotify.Write, fsnotify.Create, fsnotify.Chmod},
			EventNames: []string{"*"},
			ResetWatch: false,
		})
		watcher, err := watch.NewWatcher("html_bookmarks", watches...)
		if err != nil {
			return fmt.Errorf("setup watcher: %w", err)
		}

		// Need to track event names to detect newly added bookmark files
		watcher.TrackEventNames = true

		model.watcher = watcher
	}
	if model.watcher == nil {
		return fmt.Errorf("nothing to watch")
	}
	return nil
}

// Implements a `DumBPreLoader`
func (im *BookmarksImporter) PreLoad() ([]*gosuki.Bookmark, error) {
	var bookmarks []*gosuki.Bookmark
	result := []*gosuki.Bookmark{}

	for _, watchedPath := range model.watchedPaths {
		files, err := filepath.Glob(filepath.Join(watchedPath, "*.htm*"))
		if err != nil {
			return nil, fmt.Errorf("listing files: %w", err)
		}

		for _, file := range files {

			// skip directories
			if info, err := os.Stat(file); err != nil {
				return nil, fmt.Errorf("file stat: %w", err)
			} else if info.IsDir() {
				continue
			}
			if bookmarks, err = loadBookmarksFromHTML(file); err != nil {
				return nil, err
			}
			result = append(result, bookmarks...)
		}

	}

	go func() {
		events.TUIBus <- events.ProgressUpdateMsg{
			ID:           ImporterID,
			Instance:     nil,
			CurrentCount: uint(len(result)),
			Total:        uint(len(result)),
		}
	}()
	return result, nil
}

// Load implements watch.WatchLoader.
func (im *BookmarksImporter) Load() ([]*gosuki.Bookmark, error) {
	var err error
	var bookmarks []*gosuki.Bookmark
	result := []*gosuki.Bookmark{}

	re := regexp.MustCompile(".*html?$")

	for _, path := range model.watcher.EventNames {
		// clear out removed files from event names
		if exists, _ := utils.CheckFileExists(path); !exists {
			model.watcher.EventNames = slices.DeleteFunc(model.watcher.EventNames,
				func(p string) bool { return p == path })

			// skip this event name (file)
			continue
		}

		// only match *.html
		if match := re.Match([]byte(path)); !match || err != nil {
			continue
		}

		log.Debug("importing html bookmarks", "path", path)
		if bookmarks, err = loadBookmarksFromHTML(path); err != nil {
			return nil, err
		}
		result = append(result, bookmarks...)
	}

	for _, bk := range result {
		if err = parsing.ParseBkTags(bk); err != nil {
			return nil, fmt.Errorf("failed to parse tags: %w", err)
		}
	}
	return result, nil
}

func (im BookmarksImporter) Watch() *watch.WatchDescriptor {

	return model.watcher
}

func (im BookmarksImporter) Name() string {
	return ImporterID
}

func (im BookmarksImporter) ModInfo() modules.ModInfo {
	return modules.ModInfo{
		ID: modules.ModID(ImporterID),
		New: func() modules.Module {
			return &BookmarksImporter{}
		},
	}
}

func loadBookmarksFromHTML(filePath string) ([]*gosuki.Bookmark, error) {
	srcFile, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer srcFile.Close()

	doc, err := goquery.NewDocumentFromReader(srcFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var bookmarks []*gosuki.Bookmark
	urlsSeen := make(map[string]bool)

	doc.Find("dt>a").Each(func(_ int, a *goquery.Selection) {
		dt := a.Parent()
		dl := dt.Parent()
		h3 := dl.Parent().Find("h3").First()

		title := strings.TrimSpace(a.Text())
		url, exists := a.Attr("href")
		if !exists {
			return
		}

		// Skip if URL already seen
		if urlsSeen[url] {
			return
		}
		urlsSeen[url] = true

		// Extract tags
		tags := make([]string, 0)
		category := strings.TrimSpace(h3.Text())

		if category != "" {
			tags = append(tags, category)
		}

		bookmark := &gosuki.Bookmark{
			URL:    url,
			Title:  title,
			Tags:   tags,
			Module: ImporterID,
		}
		// fmt.Printf("%#v\n", bookmark.URL)

		bookmarks = append(bookmarks, bookmark)
	})

	return bookmarks, nil
}

type BookmarksImporterConfig struct {
	Paths []string `toml:"paths" mapstructure:"paths"`
}

func setupDefaultImportPath() []string {
	var dataDir string
	var err error

	if dataDir, err = utils.GetDataDir(); err != nil {
		log.Fatal(err)
	}

	importDir := filepath.Join(dataDir, "gosuki/imports")
	if err = utils.MkDir(importDir); err != nil {
		log.Errorf("auto import dir: %s", err)
	}

	return []string{importDir}
}

func init() {
	model = &importerModel{}
	Config = &BookmarksImporterConfig{
		Paths: setupDefaultImportPath(),
	}
	config.RegisterConfigurator(ImporterID, config.AsConfigurator(Config))
	modules.RegisterModule(&BookmarksImporter{})
}

// interface guards
var _ modules.Initializer = (*BookmarksImporter)(nil)
var _ watch.WatchLoader = (*BookmarksImporter)(nil)
var _ modules.DumbPreLoader = (*BookmarksImporter)(nil)
