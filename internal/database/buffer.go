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

package database

import (
	"fmt"

	"github.com/teris-io/shortid"

	"github.com/blob42/gosuki/pkg/tree"
)

// A Buffer is an in-memory sqlite database holding the current state of parsed
// bookmarks within a specific module. Buffers act as temporary, per-module
// storage that aggregates data before synchronizing with the Level 1 Cache
// (Cache). This decouples module processing from the global cache hierarchy,
// enabling efficient batching of updates and reducing contention. Buffers are
// ephemeral and exist only for the duration of module operations, with their
// contents periodically flushed and mereged into the Level 1 Cache to propagate
// changes upward in the two-level architecture. This design ensures minimal I/O
// overhead while maintaining consistency through checksum-based comparisons
// between cache levels.
func NewBuffer(name string) (*DB, error) {
	// add random id to buf name
	randID := shortid.MustGenerate()
	bufName := fmt.Sprintf("buffer_%s_%s", name, randID)
	// bufName := fmt.Sprintf("buffer_%s", name)
	log.Debugf("creating buffer %s", bufName)
	buffer, err := NewDB(bufName, "", DBTypeInMemoryDSN).Init()
	if err != nil {
		return nil, fmt.Errorf("could not create buffer %w", err)
	}

	err = buffer.InitSchema()
	if err != nil {
		return nil, fmt.Errorf("could initialize buffer schema %w", err)
	}

	//TEST: sqlite table locked
	buffer.Handle.SetMaxOpenConns(1)

	return buffer, nil
}

func SyncURLIndexToBuffer(urls []string, index Index, buffer *DB) {
	if buffer == nil {
		log.Error("buffer is nil")
		return
	}
	if index == nil {
		log.Error("index is nil")
		return
	}

	//OPTI: hot path
	for _, url := range urls {
		iNode, exists := index.Get(url)
		if !exists {
			log.Warnf("url does not exist in index: %s", url)
			break
		}
		node := iNode.(*Node)
		bk := node.GetBookmark()
		err := buffer.UpsertBookmark(bk)
		if err != nil {
			log.Errorf("db upsert: %s", bk.URL)
		}

	}
}

func SyncTreeToBuffer(node *Node, buffer *DB) {
	if node.Type == tree.URLNode {
		bk := node.GetBookmark()
		err := buffer.UpsertBookmark(bk)
		if err != nil {
			log.Errorf("db upsert: %s", bk.URL)
		}
	}

	if len(node.Children) > 0 {
		for _, node := range node.Children {
			SyncTreeToBuffer(node, buffer)
		}
	}
}
