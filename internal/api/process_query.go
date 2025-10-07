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

package api

import (
	"context"
	"net/http"
	"strings"

	db "github.com/blob42/gosuki/internal/database"
)

type ReqIsFuzzy struct{}

type searchQueryParts struct {
	TextQuery string
	Tags      []string
	TagCond   db.TagCond
}

func IsFuzzy(r *http.Request) bool {
	fuzzy := r.Context().Value(ReqIsFuzzy{})

	if v, ok := fuzzy.(bool); ok && v {
		return true
	}

	return false
}

// Find and add fuzzy search parameter to the request context
func trackFuzzySearch(r *http.Request) *http.Request {
	var fuzzy bool

	query := r.URL.Query().Get("query")

	if fuzzyParam := r.URL.Query().Get("fuzzy"); fuzzyParam != "" {
		fuzzy = true
	}

	// Check if the first character of query is `~`
	if len(query) > 0 && query[0] == '~' {
		fuzzy = true
	}

	rCtx := context.WithValue(r.Context(), ReqIsFuzzy{}, fuzzy)
	return r.WithContext(rCtx)
}

// ParseSearchQuery parses a search query string into its components.
// The query consists of an optional text search term followed by optional tag filters.
// The text search term matches against URL and Title fields and comes before any filters.
// Tag filters start with a colon (:) followed by either:
// 1. A comma-separated list of tags (AND logic - all tags must be present)
// 2. ":OR" followed by a comma-separated list of tag names (OR logic - any tag can match)
//
// Example queries:
//   - "golang": text query "golang" with no tags
//   - "golang :web,programming": text query "golang" with AND tags "web" and "programming"
//   - "golang :OR web,programming": text query "golang" with OR tags "web" and "programming"
func ParseSearchQuery(query string) searchQueryParts {
	var result searchQueryParts
	result.TagCond = db.TagAnd

	if !strings.Contains(query, ":") {
		result.TextQuery = strings.TrimSpace(query)
		return result
	}

	if query == ":" {
		return result
	}

	splitCount := strings.Count(query, ":")
	parts := strings.SplitN(query, ":", splitCount+1)

	textQuery := strings.Join(parts[:splitCount], ":")

	result.TextQuery = strings.TrimSpace(textQuery)

	tagPart := strings.TrimSpace(strings.Join(parts[splitCount:], ":"))

	if strings.HasPrefix(tagPart, "OR ") {
		result.TagCond = db.TagOr
		result.Tags = strings.Split(strings.TrimPrefix(tagPart, "OR "), ",")
	} else if tagPart != "" {
		result.Tags = strings.Split(tagPart, ",")
	}

	for i, tag := range result.Tags {
		result.Tags[i] = strings.TrimSpace(tag)
	}

	return result
}
