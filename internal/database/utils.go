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
	"embed"
	_ "io"

	"github.com/gchaincl/dotsql"
	"github.com/swithek/dotsqlx"
)

// Get database directory path
func GetDBDir() string {
	dbDir, err := getPlatformDBDir()
	if err != nil {
		log.Fatal(err)
	}
	return dbDir
}

// Loads a dotsql <file> and, wraps it with dotsqlx
func DotxQuery(file string) (*dotsqlx.DotSqlx, error) {
	dot, err := dotsql.LoadFromFile(file)
	if err != nil {
		return nil, err
	}

	return dotsqlx.Wrap(dot), nil
}

// Loads a dotsql from an embedded FS
func DotxQueryEmbedFS(fs embed.FS, filename string) (*dotsqlx.DotSqlx, error) {

	rawsql, err := fs.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	dot, err := dotsql.LoadFromString(string(rawsql))
	if err != nil {
		return nil, err
	}

	return dotsqlx.Wrap(dot), nil
}
