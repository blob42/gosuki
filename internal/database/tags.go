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
	"slices"
	"strings"

	"github.com/blob42/gosuki/internal/utils"
)

type Tags struct {
	delim string
	tags  []string
}

// Returns the list of tags as slice
func (t *Tags) Get() []string {
	return t.tags
}

// Reads tags from a slice of strings
func NewTags(tags []string, delim string) *Tags {
	return &Tags{delim: delim, tags: tags}
}

func (t *Tags) Add(tag string) {
	t.tags = append(t.tags, tag)
}

func (t *Tags) Extend(tags []string) *Tags {
	t.tags = utils.Extends(t.tags, tags...)
	return t
}

// Sanitize the list of tags before saving them to the DB
func (t *Tags) PreSanitize() *Tags {
	t.tags = utils.ReplaceInList(t.tags, TagSep, "--")
	return t
}

// Sorts the tags in the same order, order does not matter
func (t *Tags) Sort() *Tags {
	slices.SortFunc(t.tags, func(a, b string) int {
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})
	return t
}

// String representation of the tags.
// It can wrap the tags with the delim if wrap is true. This is done for
// compatibility with Buku DB format.
func (t Tags) String(wrap bool) string {
	if wrap {
		return delimWrap(strings.Join(t.tags, t.delim), t.delim)
	}
	return strings.Join(t.tags, t.delim)
}

// Builds a list of tags from a string as a Tags struct.
// It also removes empty tags
func TagsFromString(s, delim string) *Tags {
	tagslice := strings.Split(s, delim)
	tags := slices.DeleteFunc(tagslice, func(s string) bool {
		return s == ""
	})
	return &Tags{delim: delim, tags: tags}
}

// String representation of the tags. It wraps the tags with the delim.
func (t Tags) StringWrap() string {
	return delimWrap(strings.Join(t.tags, t.delim), t.delim)
}

// / Returns a string wrapped with the delim
func delimWrap(token string, delim string) string {
	if token == "" || strings.TrimSpace(token) == "" {
		return delim
	}

	if token[0] != delim[0] {
		token = delim + token
	}

	if token[len(token)-1] != delim[0] {
		token = token + delim
	}

	return token
}
