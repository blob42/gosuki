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

// Tag related parsing functions
package parsing

import (
	"fmt"
	"regexp"

	"github.com/blob42/gosuki"
	"github.com/blob42/gosuki/pkg/logging"
	"github.com/blob42/gosuki/pkg/tree"
)

const (
	// First group is tag
	// [named groups](https://github.com/StefanSchroeder/Golang-Regex-Tutorial/blob/master/01-chapter2.markdown)

	// Regex matching tests:

	//#start test2 #test3 elol
	//#start word with #end
	//word in the #middle of sentence
	//tags with a #dot.caracter
	//this is a end of sentence #tag
	// ReTags = `\B#(?P<tag>\w+\.?\w+)`
	ReTags = `#(?P<tag>[a-zA-Z0-9_.-]+)`

	// #tag:notify
	ReNotify = `\b(?P<tag>[a-zA-Z0-9_.-]+):notify`

	// Action tags start with @, the regex includes the @ sign
	ReActionTag = `@(?P<tag>[a-zA-Z0-9_.-]+)`
)

var log = logging.GetLogger("parse")

func stripHashTag(s string) string {
	return regexp.MustCompile(ReTags).ReplaceAllString(s, "")
}

// compelte this function
func stripActionTags(s string) string {
	return regexp.MustCompile(ReActionTag).ReplaceAllString(s, "")
}

// parseTags is a [gosuki.Hook] that extracts tags like #tag from the title of
// the bookmark or node.
// It takes an item of type *tree.Node or *gosuki.Bookmark, extracts all tags
// matching the regex pattern defined in ReTags, appends them to the item's Tags
// field, and removes the matched tags from the title. If the item is of an
// unsupported type, it returns an error.
func parseTags(item any) error {
	tagRe := regexp.MustCompile(ReTags)
	actionTagRe := regexp.MustCompile(ReActionTag)
	switch v := item.(type) {
	case *tree.Node:
		if v.Tags == nil {
			v.Tags = []string{}
		}
		processTags(tagRe, &v.Title, &v.Tags, false)
		v.Title = stripHashTag(v.Title)
		processTags(actionTagRe, &v.Title, &v.Tags, true)
		v.Title = stripActionTags(v.Title)
	case *gosuki.Bookmark:
		if v.Tags == nil {
			v.Tags = []string{}
		}
		processTags(tagRe, &v.Title, &v.Tags, false)
		v.Title = stripHashTag(v.Title)
		processTags(actionTagRe, &v.Title, &v.Tags, true)
		v.Title = stripActionTags(v.Title)
	default:
		return fmt.Errorf("unsupported type")
	}
	return nil
}

func processTags(
	regex *regexp.Regexp,
	title *string,
	tags *[]string,
	withSymbol bool,
) {

	matches := regex.FindAllStringSubmatch(*title, -1)
	for _, m := range matches {
		if withSymbol {
			*tags = append(*tags, m[0])
		} else {
			*tags = append(*tags, m[1])
		}
	}
	if len(*tags) > 0 {
		log.Tracef("[hook] found following tags: %s", *tags)
	}
}

func ParseNodeTags(n *tree.Node) error {
	return parseTags(n)
}

func ParseBkTags(b *gosuki.Bookmark) error {
	return parseTags(b)
}
