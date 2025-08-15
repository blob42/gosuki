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

// Package export implements common exporting functionnality
package export

import (
	"fmt"
	"io"

	"github.com/jmoiron/sqlx"

	"github.com/blob42/gosuki"
	db "github.com/blob42/gosuki/internal/database"
)

// export formats
const (
	// Netscape Bookmark file format
	NetscapeHTML = iota
	// Pocket HTML file format
	PocketHTML

	// Json pinboard/wallabag file format
	JSON

	// Generic RSS-XML format
	RSS
)

type BookmarksExporter struct {
	w io.Writer
	e Exporter

	// bookmark separator (ex: , for json fields)
	Separator string
}

func NewBookmarksExporter(
	e Exporter,
	w io.Writer,
) BookmarksExporter {
	return BookmarksExporter{w, e, ""}
}

func (be BookmarksExporter) ExportBookmarks(bookmarks []*gosuki.Bookmark) error {
	return be.e.ExportBookmarks(bookmarks, be.w)
}

func (be BookmarksExporter) ExportFromRows(rows *sqlx.Rows) error {
	var err error
	var rawBook db.RawBookmark

	if err = be.e.WriteHeader(be.w); err != nil {
		return err
	}
	first := true
	for rows.Next() {
		if err = rows.StructScan(&rawBook); err != nil {
			return fmt.Errorf("scanning book: %w", err)
		}

		if !first && be.Separator != "" {
			if _, err = be.w.Write([]byte(be.Separator)); err != nil {
				return err
			}
		} else {
			first = false
		}

		marshaled := be.e.MarshalBookmark(rawBook.AsBookmark())
		if _, err = be.w.Write(marshaled); err != nil {
			return err
		}
	}

	if err = be.e.WriteFooter(be.w); err != nil {
		return err
	}
	return nil
}

type Exporter interface {
	ExportBookmarks(bookmarks []*gosuki.Bookmark, w io.Writer) error
	MarshalBookmark(*gosuki.Bookmark) []byte
	WriteHeader(w io.Writer) error
	WriteFooter(w io.Writer) error
}
