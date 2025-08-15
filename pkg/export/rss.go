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
	"time"

	"github.com/blob42/gosuki"
)

type RSSXMLExporter struct{}

func (rs *RSSXMLExporter) WriteHeader(w io.Writer) error {
	_, err := fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Bookmarks</title>
    <link>https://example.com/bookmarks</link>
    <description>Exported bookmarks from GoSuki</description>`)
	return err
}

func (rs *RSSXMLExporter) WriteFooter(w io.Writer) error {
	_, err := fmt.Fprintf(w, `  </channel>
</rss>`)
	return err
}

func (rs *RSSXMLExporter) ExportBookmarks(bookmarks []*gosuki.Bookmark, w io.Writer) error {
	var err error

	if err = rs.WriteHeader(w); err != nil {
		return err
	}

	for _, book := range bookmarks {
		if _, err = w.Write(rs.MarshalBookmark(book)); err != nil {
			return err
		}
	}

	return rs.WriteFooter(w)
}

func (rs *RSSXMLExporter) MarshalBookmark(book *gosuki.Bookmark) []byte {
	pubDate := time.Unix(int64(book.Modified), 0).Format("Mon, 02 Jan 2006 15:04:05 -0700")

	return []byte(fmt.Appendf([]byte{}, `    <item>
      <title><![CDATA[%s]]></title>
      <link>%s</link>
      <guid>%s</guid>
      <pubDate>%s</pubDate>
    </item>`,
		html.EscapeString(book.Title),
		html.EscapeString(book.URL),
		book.Xhsum,
		pubDate))
}
