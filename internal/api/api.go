// Copyright (c) 2025 Chakib Ben Ziane <contact@blob42.xyz>  and [`gosuki` contributors](https://github.com/blob42/gosuki/graphs/contributors).
// All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This file is part of GoSuki.
//
// GoSuki is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// GoSuki is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with gosuki.  If not, see <http://www.gnu.org/licenses/>.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/blob42/gosuki"
	db "github.com/blob42/gosuki/internal/database"
)

type Bookmark = gosuki.Bookmark
type RawBookmark = db.RawBookmark

type Payload struct {
	Total   uint `json:"total"`
	Page    int  `json:"page"`
	PerPage int  `json:"per_page"`
	Result  any  `json:"result"`
}

type ReqIsFuzzy struct{}
type ResetPage struct{}

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

func GetPaginationParams(r *http.Request) *db.PaginationParams {
	pageParams := db.DefaultPagination()
	resetPage := r.Context().Value(ResetPage{})

	if v, ok := resetPage.(bool); ok && v {
		pageParams.Page = 1
	} else if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			pageParams.Page = page
		}
	}
	if ppageStr := r.URL.Query().Get("per_page"); ppageStr != "" {
		if pPage, err := strconv.Atoi(ppageStr); err == nil {
			pageParams.Size = pPage
		}
	}

	return pageParams
}

func GetAPIBookmarks(w http.ResponseWriter, r *http.Request) {
	bookmarks, total, err := GetBookmarks(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	pageParams := GetPaginationParams(r)

	payload := Payload{
		Total:   total,
		Page:    pageParams.Page,
		PerPage: pageParams.Size,
		Result:  bookmarks,
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func GetBookmarks(r *http.Request) ([]*gosuki.Bookmark, uint, error) {
	var qResult *db.QueryResult
	var err error

	r = trackFuzzySearch(r)

	urlQuery := r.URL.Query()

	if tag := chi.URLParam(r, "tag"); tag != "" {
		urlQuery.Add("tag", tag)
	}

	// Handle query by tag/query
	query := urlQuery.Get("query")
	tag := urlQuery.Get("tag")

	// Check if the first character of query is `~`
	if len(query) > 0 && query[0] == '~' {
		query = query[1:] // Trim the first character
	}

	pageParams := GetPaginationParams(r)
	var (
		searchByQuery = query != ""
		searchByTag   = tag != ""
	)

	// Search with query AND tags
	if searchByQuery && searchByTag {
		if strings.Contains(tag, ",") {
			// Query with multiple tags
			qResult, err = db.QueryBookmarksByTags(
				r.Context(),
				query,
				strings.Split(tag, ","),
				db.TagAnd, // AND tags
				IsFuzzy(r),
				pageParams,
			)
		} else {
			qResult, err = db.QueryBookmarksByTag(r.Context(), query, tag, IsFuzzy(r), pageParams)
		}

		// Tag only search mode
	} else if searchByTag {
		//query with many tags
		if strings.Contains(tag, ",") {
			qResult, err = db.BookmarksByTags(
				r.Context(),
				strings.Split(tag, ","),
				db.TagAnd,
				pageParams,
			)
		} else {
			qResult, err = db.BookmarksByTag(r.Context(), tag, pageParams)
		}
	} else if searchByQuery {
		qResult, err = db.QueryBookmarks(r.Context(), query, IsFuzzy(r), pageParams)

		// No filter mode, list all bookmarks paginated
	} else {
		qResult, err = db.ListBookmarks(r.Context(), pageParams)
	}

	if err != nil {
		return nil, 0, fmt.Errorf("database query failed: %w", err)
	}

	return qResult.Bookmarks, qResult.Total, nil
}
