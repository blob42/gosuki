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
	"testing"

	db "github.com/blob42/gosuki/internal/database"
)

func TestParseSearchQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected searchQueryParts
	}{
		{
			name:  "no tags",
			query: "golang",
			expected: searchQueryParts{
				TextQuery: "golang",
				Tags:      nil,
				TagCond:   db.TagAnd,
			},
		},
		{
			name:  "text only with leading/trailing spaces",
			query: "  golang  ",
			expected: searchQueryParts{
				TextQuery: "golang",
				Tags:      nil,
				TagCond:   db.TagAnd,
			},
		},
		{
			name:  "simple AND tags",
			query: "golang :web,programming",
			expected: searchQueryParts{
				TextQuery: "golang",
				Tags:      []string{"web", "programming"},
				TagCond:   db.TagAnd,
			},
		},
		{
			name:  "AND tags with spaces",
			query: "golang :web, programming, go",
			expected: searchQueryParts{
				TextQuery: "golang",
				Tags:      []string{"web", "programming", "go"},
				TagCond:   db.TagAnd,
			},
		},
		{
			name:  "OR tags with OR prefix",
			query: "golang :OR web,programming",
			expected: searchQueryParts{
				TextQuery: "golang",
				Tags:      []string{"web", "programming"},
				TagCond:   db.TagOr,
			},
		},
		{
			name:  "OR spaced tags with OR prefix",
			query: "golang :OR web programming",
			expected: searchQueryParts{
				TextQuery: "golang",
				Tags:      []string{"web programming"},
				TagCond:   db.TagOr,
			},
		},
		{
			name:  "OR tags with spaces",
			query: "golang :OR web, programming, rust",
			expected: searchQueryParts{
				TextQuery: "golang",
				Tags:      []string{"web", "programming", "rust"},
				TagCond:   db.TagOr,
			},
		},
		{
			name:  "OR tags with leading spaces",
			query: "golang :OR  web, programming",
			expected: searchQueryParts{
				TextQuery: "golang",
				Tags:      []string{"web", "programming"},
				TagCond:   db.TagOr,
			},
		},
		{
			name:  "mixed case tags",
			query: "golang :OR Web,Programming",
			expected: searchQueryParts{
				TextQuery: "golang",
				Tags:      []string{"Web", "Programming"},
				TagCond:   db.TagOr,
			},
		},
		{
			name:  "empty text query",
			query: ":web,programming",
			expected: searchQueryParts{
				TextQuery: "",
				Tags:      []string{"web", "programming"},
				TagCond:   db.TagAnd,
			},
		},
		{
			name:  "empty tag list",
			query: "golang :",
			expected: searchQueryParts{
				TextQuery: "golang",
				Tags:      nil,
				TagCond:   db.TagAnd,
			},
		},
		{
			name:  "no text query with OR tags",
			query: ":OR web,programming",
			expected: searchQueryParts{
				TextQuery: "",
				Tags:      []string{"web", "programming"},
				TagCond:   db.TagOr,
			},
		},
		{
			name:  "multiple colons in text",
			query: "golang:awesome :OR web,programming",
			expected: searchQueryParts{
				TextQuery: "golang:awesome",
				Tags:      []string{"web", "programming"},
				TagCond:   db.TagOr,
			},
		},
		{
			name:  "colon at end",
			query: "golang:",
			expected: searchQueryParts{
				TextQuery: "golang",
				Tags:      nil,
				TagCond:   db.TagAnd,
			},
		},
		{
			name:  "only colon",
			query: ":",
			expected: searchQueryParts{
				TextQuery: "",
				Tags:      nil,
				TagCond:   db.TagAnd,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSearchQuery(tt.query)

			if result.TextQuery != tt.expected.TextQuery {
				t.Errorf("TextQuery = %q, want %q", result.TextQuery, tt.expected.TextQuery)
			}

			if result.TagCond != tt.expected.TagCond {
				t.Errorf("TagCond = %v, want %v", result.TagCond, tt.expected.TagCond)
			}

			if len(result.Tags) != len(tt.expected.Tags) {
				t.Errorf("Tags length = %d, want %d", len(result.Tags), len(tt.expected.Tags))
				return
			}

			for i, expectedTag := range tt.expected.Tags {
				if result.Tags[i] != expectedTag {
					t.Errorf("Tags[%d] = %q, want %q", i, result.Tags[i], expectedTag)
				}
			}
		})
	}
}

func TestParseSearchQueryEdgeCases(t *testing.T) {
	// Test that the function handles edge cases properly
	tests := []struct {
		name     string
		query    string
		wantText string
	}{
		{
			name:     "empty string",
			query:    "",
			wantText: "",
		},
		{
			name:     "only spaces",
			query:    "   ",
			wantText: "",
		},
		{
			name:     "single colon",
			query:    ":",
			wantText: "",
		},
		{
			name:     "colon with space",
			query:    ": ",
			wantText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSearchQuery(tt.query)
			if result.TextQuery != tt.wantText {
				t.Errorf("TextQuery = %q, want %q", result.TextQuery, tt.wantText)
			}
		})
	}
}
