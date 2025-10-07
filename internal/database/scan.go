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

package database

import (
	"database/sql/driver"
	"fmt"
	"strconv"

	"github.com/blob42/gosuki"
	"github.com/gofrs/uuid"
)

type xxhashsum uint64
type UUID uuid.UUID

// Implements the [sql.Scanner] interface.
func (hash *xxhashsum) Scan(value any) error {
	if value == nil {
		*hash = 0
		return nil
	}

	switch s := value.(type) {
	case uint64:
		*hash = xxhashsum(s)
		return nil
	case string:
		if s == "" {
			*hash = xxhashsum(0)
			return nil
		}
		val, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse uint64 from string \"%s\"", value)
		}
		*hash = xxhashsum(val)
	case []byte:
		val, err := strconv.ParseUint(string(s), 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse uint64 from []byte %v", value)
		}
		*hash = xxhashsum(val)
	default:
		return fmt.Errorf("cannot convert to uint64 %v", value)
	}
	return nil
}

func (nodeID *UUID) Scan(value any) error {
	if value == nil {
		*nodeID = UUID(uuid.Nil)
		return nil
	}

	if bytes, ok := value.([]byte); ok {
		if u, err := uuid.FromBytes(bytes); err != nil {
			return fmt.Errorf("could not parse uuid.UUID: %s", err)
		} else {
			*nodeID = UUID(u)
		}

	} else {
		return fmt.Errorf("cannot parse uuid.UUID from %v", value)
	}

	return nil
}

func (nodeID UUID) Value() (driver.Value, error) {
	if nodeID == UUID(uuid.Nil) {
		return nil, nil
	}
	return uuid.UUID(nodeID).Bytes(), nil
}

type RawBookmarks []*RawBookmark

func (raw RawBookmark) AsBookmark() *gosuki.Bookmark {
	return &gosuki.Bookmark{
		URL:      raw.URL,
		Title:    raw.Metadata,
		Tags:     tagsFromString(raw.Tags, TagSep).Get(),
		Desc:     raw.Desc,
		Module:   raw.Module,
		Modified: raw.Modified,
		Xhsum:    raw.XHSum,
	}
}

func (raws RawBookmarks) AsBookmarks() []*gosuki.Bookmark {
	res := make([]*gosuki.Bookmark, 0, len(raws))
	for _, raw := range raws {
		res = append(res, raw.AsBookmark())
	}
	return res
}

type RawBookmark struct {
	ID  uint64
	URL string `db:"URL"`

	// Usually used for the bookmark title
	Metadata string

	Tags string
	Desc string

	// Last modified
	Modified uint64

	// kept for buku compat, not used for now
	Flags int

	Module string

	// currently not used
	XHSum string

	// lamport clock
	Version uint64

	// Node that made the change
	NodeID UUID `db:"node_id"`
}
