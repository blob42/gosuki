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

package export

import (
	"fmt"
	"html"
	"io"
	"strings"

	"github.com/blob42/gosuki"
	db "github.com/blob42/gosuki/internal/database"
)

type NetscapeHTMLExporter struct{}

func (ns *NetscapeHTMLExporter) WriteHeader(w io.Writer) error {
	_, err := fmt.Fprintf(w, `<!DOCTYPE NETSCAPE-Bookmark-file-1>
<META HTTP-EQUIV="Content-Type" CONTENT="text/html; charset=UTF-8">
<TITLE>Bookmarks</TITLE>
<H1>Bookmarks</H1>
<DL>`)
	return err
}

func (ns *NetscapeHTMLExporter) WriteFooter(w io.Writer) error {
	_, err := fmt.Fprintf(w, "</DL>\n")
	return err
}

func (ns *NetscapeHTMLExporter) ExportBookmarks(bookmarks []*gosuki.Bookmark, w io.Writer) error {
	var err error

	if err = ns.WriteHeader(w); err != nil {
		return err
	}

	for _, book := range bookmarks {
		if _, err = w.Write(ns.MarshalBookmark(book)); err != nil {
			return err
		}
	}

	return ns.WriteFooter(w)
}

func (ns *NetscapeHTMLExporter) MarshalBookmark(book *gosuki.Bookmark) []byte {
	return fmt.Appendf([]byte{}, `    <DT><A HREF="%s" TAGS="%s" ADD_DATE="%d" LAST_MODIFIED="%d">%s</A>
`,
		html.EscapeString(book.URL),

		// this is not conform to netscape export format, but we still save tags here
		strings.Join(book.Tags, db.TagSep),

		book.Modified,
		book.Modified,

		html.EscapeString(book.Title),
	)
}
