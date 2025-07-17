// Copyright (c) 2024-2025-2025-2025-2025 Chakib Ben Ziane <contact@blob42.xyz>  and [`gosuki` contributors](https://github.com/blob42/gosuki/graphs/contributors).
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

package database

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/blob42/gosuki"
)

const (
	WhereQueryBookmarks = `
	URL like '%%%s%%' OR metadata like '%%%s%%' OR tags like '%%%s%%'
	`

	WhereQueryBookmarksFuzzy = `
	fuzzy('%s', URL) OR fuzzy('%s', metadata) OR fuzzy('%s', tags)
	`

	WhereQueryBookmarksByTag = `
		(URL LIKE '%%%s%%' OR metadata LIKE '%%%s%%') AND tags LIKE '%%%s%%'
	`
	WhereQueryBookmarksByTagFuzzy = `
		(fuzzy('%s', URL) OR fuzzy('%s', metadata)) AND tags LIKE '%%%s%%'
	`

	QQueryPaginate = ` LIMIT %d OFFSET %d`
)

type xxhashsum uint64

// Implements the [sql.Scanner] interface.
func (hash *xxhashsum) Scan(value any) error {
	if value == nil {
		*hash = 0
		return nil
	}
	if s, ok := value.(uint64); ok {
		*hash = xxhashsum(s)
		return nil
	} else if s, ok := value.(string); ok {
		val, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse uint64 from string %v", value)
		}
		*hash = xxhashsum(val)
	} else if s, ok := value.([]byte); ok {
		val, err := strconv.ParseUint(string(s), 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse uint64 from []byte %v", value)
		}
		*hash = xxhashsum(val)
	} else {
		return fmt.Errorf("cannot convert to uint64 %v", value)
	}
	return nil
}

type RawBookmarks []*RawBookmark

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
}

type PaginationParams struct {
	Page int
	Size int
}

type QueryResult struct {
	Bookmarks []*gosuki.Bookmark
	Total     uint
}

func DefaultPagination() *PaginationParams {
	return &PaginationParams{1, 50}
}

func (raws RawBookmarks) AsBookmarks() []*gosuki.Bookmark {
	res := []*Bookmark{}
	for _, raw := range raws {
		tags := TagsFromString(raw.Tags, TagSep)
		res = append(res, &Bookmark{
			URL:    raw.URL,
			Title:  raw.Metadata,
			Tags:   tags.Get(),
			Desc:   raw.Desc,
			Module: raw.Module,
		})
	}

	return res
}

func QueryBookmarksByTag(
	ctx context.Context,
	query,
	tag string,
	fuzzy bool,
	pagination *PaginationParams,
) (*QueryResult, error) {
	query = strings.TrimSpace(query)
	tag = strings.TrimSpace(tag)

	if pagination == nil {
		return nil, errors.New("nil: *PaginationParams")
	}

	if tag == "" || query == "" {
		return nil, errors.New("cannot use empty query or tags")
	}

	sqlQuery := buildSelectQuery(query, fuzzy, tag, pagination)

	rawBooks := RawBookmarks{}
	err := DiskDB.Handle.SelectContext(ctx, &rawBooks, sqlQuery)
	if err != nil {
		return nil, err
	}

	var total uint
	err = DiskDB.Handle.GetContext(ctx, &total,
		fmt.Sprintf(buildCountQuery(tag, fuzzy), query, query, query))
	if err != nil {
		return nil, err
	}

	return &QueryResult{rawBooks.AsBookmarks(), total}, nil
}

func QueryBookmarks(
	ctx context.Context,
	query string,
	fuzzy bool,
	pagination *PaginationParams,
) (*QueryResult, error) {

	if query == "" {
		return nil, errors.New("cannot use empty query or tags")
	}

	sqlQuery := buildSelectQuery(query, fuzzy, "", pagination)

	rawBooks := RawBookmarks{}
	err := DiskDB.Handle.SelectContext(ctx, &rawBooks, sqlQuery)
	if err != nil {
		return nil, err
	}

	var total uint
	err = DiskDB.Handle.GetContext(ctx, &total,
		fmt.Sprintf(buildCountQuery("", fuzzy), query, query, query))
	if err != nil {
		return nil, err
	}

	return &QueryResult{rawBooks.AsBookmarks(), total}, nil
}

func BookmarksByTag(
	ctx context.Context,
	tag string,
	pagination *PaginationParams,
) (*QueryResult, error) {
	query := "SELECT * FROM gskbookmarks WHERE"
	tagsCondition := ""
	if len(tag) > 0 {
		tagsCondition = fmt.Sprintf(" tags LIKE '%%%s%%'", tag)
	} else {
		return nil, errors.New("empty tag provided")
	}

	query = query + " (" + tagsCondition + ")"
	query += fmt.Sprintf(" "+QQueryPaginate, pagination.Size, (pagination.Page-1)*pagination.Size)

	rawBooks := RawBookmarks{}
	err := DiskDB.Handle.SelectContext(ctx, &rawBooks, query)
	if err != nil {
		return nil, err
	}

	var count uint
	err = DiskDB.Handle.GetContext(
		ctx,
		&count,
		fmt.Sprintf("SELECT COUNT(*) FROM gskbookmarks WHERE %s", tagsCondition),
	)
	if err != nil {
		return nil, err
	}

	return &QueryResult{rawBooks.AsBookmarks(), count}, nil
}

func ListBookmarks(
	ctx context.Context,
	pagination *PaginationParams,
) (*QueryResult, error) {
	rawBooks := RawBookmarks{}
	err := DiskDB.Handle.SelectContext(
		ctx,
		&rawBooks,
		fmt.Sprintf("SELECT * FROM gskbookmarks LIMIT %d OFFSET %d",
			pagination.Size,
			(pagination.Page-1)*pagination.Size,
		),
	)
	if err != nil {
		return nil, err
	}

	total, err := CountTotalBookmarks(ctx)
	if err != nil {
		return nil, fmt.Errorf("counting urls: %w", err)
	}

	return &QueryResult{rawBooks.AsBookmarks(), total}, nil
}

func CountTotalBookmarks(ctx context.Context) (uint, error) {
	var count uint
	err := DiskDB.Handle.GetContext(ctx, &count, "SELECT COUNT(*) FROM gskbookmarks LIMIT 1")
	if err != nil {
		return 0, err
	}
	return count, nil
}

func buildSelectQuery(
	query string,
	fuzzy bool,
	tag string,
	pagination *PaginationParams,
) string {

	if pagination == nil {
		log.Fatal("nil pagination")
	}

	sqlPrelude := `
		SELECT URL, metadata, tags, module
		FROM gskbookmarks
		WHERE 
	`

	sqlQuery := fmt.Sprintf(
		"%s %s %s",
		sqlPrelude,
		buildWhereClause(tag, fuzzy),
		QQueryPaginate,
	)

	if tag == "" {
		tag = query
	}

	return fmt.Sprintf(
		sqlQuery,
		query,
		query,
		tag,
		pagination.Size,
		(pagination.Page-1)*pagination.Size,
	)
}

func buildWhereClause(tag string, fuzzy bool) string {

	sqlQuery := WhereQueryBookmarks

	// query by tag
	if len(tag) > 0 && !fuzzy {
		sqlQuery = WhereQueryBookmarksByTag
	} else if len(tag) > 0 && fuzzy {
		sqlQuery = WhereQueryBookmarksByTagFuzzy
	} else if fuzzy {
		sqlQuery = WhereQueryBookmarksFuzzy
	}

	return sqlQuery
}

func buildCountQuery(tag string, fuzzy bool) string {
	query := fmt.Sprintf(`SELECT COUNT(*) FROM gskbookmarks WHERE %s LIMIT 1`,
		buildWhereClause(tag, fuzzy),
	)
	return query
}
