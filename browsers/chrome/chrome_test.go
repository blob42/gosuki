// Copyright (c) 2023 Chakib Ben Ziane <contact@blob42.xyz>  and [`gosuki` contributors](https://github.com/blob42/gosuki/graphs/contributors).
// All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This file is part of GoSuki.
//
// GoSuki is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// GoSuki is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with gosuki.  If not, see <http://www.gnu.org/licenses/>.
package chrome

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/blob42/gosuki/internal/database"
	"github.com/blob42/gosuki/internal/index"
	"github.com/blob42/gosuki/internal/utils"
	"github.com/blob42/gosuki/pkg/browsers"
	"github.com/blob42/gosuki/pkg/logging"
	"github.com/blob42/gosuki/pkg/modules"
	"github.com/blob42/gosuki/pkg/parsing"
	"github.com/blob42/gosuki/pkg/profiles"
	"github.com/blob42/gosuki/pkg/tree"
)

const statePath = "testdata/Local State"

var ch Chrome

func setupChrome() {
	bufDB, err := database.NewBuffer("chrome_test")
	if err != nil {
		panic(err)
	}
	ch = Chrome{
		ChromeConfig: &ChromeConfig{
			BrowserConfig: &modules.BrowserConfig{
				Name:     "chrome",
				BaseDir:  "",
				BkDir:    "testdata",
				BkFile:   "Bookmarks",
				BufferDB: bufDB,
				URLIndex: index.NewIndex(),
				NodeTree: &tree.Node{
					Title:  RootNodeName,
					Parent: nil,
					Type:   tree.RootNode,
				},
				UseFileWatcher: true,
				UseHooks:       []string{},
			},
		},
		Counter: &parsing.BrowserCounter{},
	}
}

func TestMain(m *testing.M) {
	database.RegisterSqliteHooks()

	cacheDB, err := database.NewDB(database.CacheName, "", database.DBTypeCacheDSN).Init()
	if err != nil {
		log.Fatal(err)
	}

	database.Cache = &database.CacheDB{DB: cacheDB}

	setupChrome()
	exitVal := m.Run()
	os.Exit(exitVal)
}

var blackholeState *StateData

func TestLoadLocalState(t *testing.T) {
	fullPath, err := utils.ExpandPath(statePath)
	if err != nil {
		t.Fatal(err)
	}
	blackholeState, err = loadLocalState(fullPath)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetProfiles(t *testing.T) {
	var needle *profiles.Profile

	browsers.AddBrowserDef(browsers.ChromeBrowser(BrowserName, "testdata", "", ""))
	ch := &Chrome{}
	profiles, err := ch.GetProfiles(BrowserName)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(profiles), "wrong number of profiles found")

	for _, profile := range profiles {
		if profile.ID == "Default" {
			needle = profile
			break
		}
	}
	assert.NotNil(t, needle, "No profile with ID 'Default' found")
}

func TestRun(t *testing.T) {
	logging.SetLogLevel(-1)
	ch.Run()

	// dummy google Bookmarks file url count
	assert.EqualValues(t, 2007, int(ch.URLCount()), "wrong # of parsed urls")

	// 2007 urls and 1909 folders
	assert.EqualValues(t, 2007+1909, int(ch.NodeCount()), "wrong # of parsed nodes")

}

func TestPreCount(t *testing.T) {
	assert.NoError(t, ch.PreLoad(&modules.Context{}), "error preloading bookmarks")
	total := ch.Total()
	assert.EqualValues(t, 2007, int(total), "wrong # of url count")
}

func BenchmarkRun(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ch.Run()
	}
}
