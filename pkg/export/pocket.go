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
)

type PocketHTMLExporter struct{}

func (pe *PocketHTMLExporter) WriteHeader(w io.Writer) error {
	_, err := fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Pocket Bookmarks</title></head>
<body>
<ul id="pocket-bookmarks">`)
	return err
}

func (pe *PocketHTMLExporter) WriteFooter(w io.Writer) error {
	_, err := fmt.Fprintf(w, `</ul>
</body>
</html>`)
	return err
}

func (pe *PocketHTMLExporter) ExportBookmarks(bookmarks []*gosuki.Bookmark, w io.Writer) error {
	var err error

	if err = pe.WriteHeader(w); err != nil {
		return err
	}

	for _, book := range bookmarks {
		if _, err = w.Write(pe.MarshalBookmark(book)); err != nil {
			return err
		}
	}

	return pe.WriteFooter(w)
}

func (pe *PocketHTMLExporter) MarshalBookmark(book *gosuki.Bookmark) []byte {
	escapedURL := html.EscapeString(book.URL)
	escapedTitle := html.EscapeString(book.Title)
	tagStr := strings.Join(book.Tags, ",")
	return fmt.Appendf([]byte{}, `<li><a href="%s" time_added="%d" tags="%s">%s</a></li>
`,
		escapedURL,
		book.Modified,
		tagStr,
		escapedTitle)
}
