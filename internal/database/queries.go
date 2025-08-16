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
	"strings"

	"github.com/blob42/gosuki"
	sqlite3 "github.com/mattn/go-sqlite3"
)

const (
	WhereQueryBookmarks = `
	URL like '%%%s%%' OR metadata like '%%%s%%' OR LOWER(tags) like '%%%s%%'
	`

	WhereQueryBookmarksFuzzy = `
	fuzzy('%s', URL) OR fuzzy('%s', metadata) OR fuzzy('%s', tags)
	`

	WhereQueryBookmarksByTag = `
		(URL LIKE '%%%s%%' OR metadata LIKE '%%%s%%') AND LOWER(tags) LIKE '%%%s%%'
	`
	WhereQueryBookmarksByTagFuzzy = `
		(fuzzy('%s', URL) OR fuzzy('%s', metadata)) AND LOWER(tags) LIKE '%%%s%%'
	`

	QQueryPaginate = ` LIMIT %d OFFSET %d`
)

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

func QueryBookmarksByTags(
	ctx context.Context,
	query string,
	tags []string,
	cond TagCond,
	fuzzy bool,
	pagination *PaginationParams,
) (*QueryResult, error) {
	if len(tags) == 0 {
		return nil, errors.New("empty tags provided")
	}

	if pagination == nil {
		return nil, errors.New("nil: *PaginationParams")
	}

	// build the WHERE clause
	whereClause := buildWhereClauseForManyTags(query, tags, cond, fuzzy)
	log.Trace(whereClause)

	sqlQuery := fmt.Sprintf(
		"SELECT URL, metadata, tags, module FROM gskbookmarks WHERE %s "+QQueryPaginate,
		whereClause,
		pagination.Size,
		(pagination.Page-1)*pagination.Size,
	)

	rawBooks := RawBookmarks{}
	err := DiskDB.Handle.SelectContext(ctx, &rawBooks, sqlQuery)
	if err != nil {
		return nil, err
	}

	countQuery := fmt.Sprintf(
		"SELECT COUNT(*) FROM gskbookmarks WHERE %s LIMIT 1",
		whereClause,
	)

	var total uint
	err = DiskDB.Handle.GetContext(ctx, &total, countQuery)
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
		tagsCondition = fmt.Sprintf(" LOWER(tags) LIKE '%%%s%%'", strings.ToLower(tag))
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

type TagCond int

const (
	TagAnd = iota
	TagOr
)

func BookmarksByTags(
	ctx context.Context,
	tags []string,
	cond TagCond,
	pagination *PaginationParams,
) (*QueryResult, error) {
	if len(tags) == 0 {
		return nil, errors.New("empty tags provided")
	}

	query := "SELECT * FROM gskbookmarks WHERE"
	conditions := make([]string, 0, len(tags))

	for _, tag := range tags {
		if len(tag) > 0 {
			conditions = append(
				conditions,
				fmt.Sprintf("LOWER(tags) LIKE '%%%s%%'", strings.ToLower(tag)),
			)
		}
	}

	if len(conditions) == 0 {
		return nil, errors.New("no valid tags provided")
	}

	var joinOperator string
	if cond == TagAnd {
		joinOperator = " AND "
	} else {
		joinOperator = " OR "
	}

	query = query + " (" + strings.Join(conditions, joinOperator) + ")"
	query += fmt.Sprintf(" "+QQueryPaginate, pagination.Size, (pagination.Page-1)*pagination.Size)

	rawBooks := RawBookmarks{}
	err := DiskDB.Handle.SelectContext(ctx, &rawBooks, query)
	if err != nil {
		return nil, err
	}

	var count uint
	countQuery := "SELECT COUNT(*) FROM gskbookmarks WHERE"
	countQuery = countQuery + " (" + strings.Join(conditions, joinOperator) + ")"
	err = DiskDB.Handle.GetContext(ctx, &count, countQuery)
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

// CountTotalBookmarks counts total bookmarks from disk
func CountTotalBookmarks(ctx context.Context) (uint, error) {
	return DiskDB.TotalBookmarks(ctx)
}

func (db *DB) TotalBookmarks(ctx context.Context) (uint, error) {
	var count uint

	if db == nil || db.Handle == nil {
		return 0, nil
	}
	err := db.Handle.GetContext(ctx, &count, "SELECT COUNT(*) FROM gskbookmarks LIMIT 1")
	if err != nil {
		if sqlErr, ok := err.(sqlite3.Error); ok && sqlErr.Code == sqlite3.ErrLocked {
			return 0, nil
		}
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

func buildWhereClauseForManyTags(
	query string,
	tags []string,
	cond TagCond,
	fuzzy bool,
) string {
	conditions := make([]string, 0)
	tagsConditions := make([]string, 0)

	// query conditions
	if query != "" {
		trimmedQuery := strings.TrimSpace(query)
		if fuzzy {
			conditions = append(
				conditions,
				fmt.Sprintf(
					"fuzzy('%s', URL) OR fuzzy('%s', metadata)",
					trimmedQuery,
					trimmedQuery,
				),
			)
		} else {
			conditions = append(
				conditions,
				fmt.Sprintf(
					"URL like '%%%s%%' OR metadata like '%%%s%%'",
					trimmedQuery,
					trimmedQuery,
				),
			)
		}
	}

	// tag conditions
	for _, tag := range tags {
		if tag != "" {
			trimmedTag := strings.TrimSpace(tag)
			if fuzzy {
				tagsConditions = append(
					tagsConditions,
					fmt.Sprintf("fuzzy('%s', tags)", trimmedTag),
				)
			} else {
				tagsConditions = append(
					tagsConditions,
					fmt.Sprintf(
						"LOWER(tags) like '%%%s%%'",
						strings.ToLower(trimmedTag),
					),
				)
			}
		}
	}

	tagJoinOperator := " OR "
	if cond == TagAnd {
		tagJoinOperator = " AND "
	}

	conditionsStr := "1=1"
	if len(conditions) > 0 {
		conditionsStr = strings.Join(conditions, " AND ")
	}

	tagsStr := "1=1"
	if len(tagsConditions) > 0 {
		tagsStr = strings.Join(tagsConditions, tagJoinOperator)
	}

	// we use AND because Query is always searched and tags are for filtering
	return fmt.Sprintf("( %s ) AND ( %s )", conditionsStr, tagsStr)
}
