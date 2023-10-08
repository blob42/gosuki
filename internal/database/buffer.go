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

package database

import (
	"fmt"

	"github.com/teris-io/shortid"

	"github.com/blob42/gosuki/pkg/tree"
)



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
			log.Warningf("url does not exist in index: %s", url)
			break
		}
		node := iNode.(*Node)
		bk := node.GetBookmark()
		buffer.UpsertBookmark(bk)
	}
}

func SyncTreeToBuffer(node *Node, buffer *DB) {
	if node.Type == tree.URLNode {
		bk := node.GetBookmark()
		buffer.UpsertBookmark(bk)
	}

	if len(node.Children) > 0 {
		for _, node := range node.Children {
			SyncTreeToBuffer(node, buffer)
		}
	}
}
