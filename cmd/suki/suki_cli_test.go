package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSortFlag_Empty(t *testing.T) {
	sortBy, sortAsc := parseSortFlag("")
	require.Equal(t, "", sortBy, "empty input should return empty sortBy")
	require.False(t, sortAsc)
}

func TestParseSortFlag_FieldOnly(t *testing.T) {
	tests := []struct {
		input    string
		sortBy   string
		sortAsc  bool
	}{
		{"modified", "modified", false}, // default DESC
		{"title", "title", false},
		{"url", "url", false},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			sortBy, sortAsc := parseSortFlag(tc.input)
			require.Equal(t, tc.sortBy, sortBy)
			require.Equal(t, tc.sortAsc, sortAsc)
		})
	}
}

func TestParseSortFlag_WithDirection(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		sortBy  string
		sortAsc bool
	}{
		{"modified:asc", "modified:asc", "modified", true},
		{"modified:ASC", "modified:ASC", "modified", true},
		{"modified:desc", "modified:desc", "modified", false},
		{"modified:DESC", "modified:DESC", "modified", false},
		{"title:asc", "title:asc", "title", true},
		{"title:desc", "title:desc", "title", false},
		{"url:asc", "url:asc", "url", true},
		{"url:desc", "url:desc", "url", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sortBy, sortAsc := parseSortFlag(tc.input)
			require.Equal(t, tc.sortBy, sortBy)
			require.Equal(t, tc.sortAsc, sortAsc)
		})
	}
}

func TestParseSortFlag_Whitespace(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		sortBy  string
		sortAsc bool
	}{
		{"leading space", "  modified", "modified", false},
		{"trailing space", "modified  ", "modified", false},
		{"both spaces", "  title  ", "title", false},
		{"space before colon", "modified :asc", "modified", true},
		{"space after colon", "modified: asc", "modified", true},
		{"spaces around", "  url : desc  ", "url", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sortBy, sortAsc := parseSortFlag(tc.input)
			require.Equal(t, tc.sortBy, sortBy)
			require.Equal(t, tc.sortAsc, sortAsc)
		})
	}
}

func TestParseSortFlag_WhitespaceOnly(t *testing.T) {
	sortBy, sortAsc := parseSortFlag("   ")
	require.Equal(t, "", sortBy, "whitespace-only should return empty sortBy")
	require.False(t, sortAsc)
}

func TestParseSortFlag_InvalidDirection(t *testing.T) {
	// Unknown direction defaults to DESC
	sortBy, sortAsc := parseSortFlag("modified:reverse")
	require.Equal(t, "modified", sortBy)
	require.False(t, sortAsc, "unknown direction should default to DESC")
}

func TestParseSortFlag_JustColon(t *testing.T) {
	sortBy, sortAsc := parseSortFlag(":asc")
	require.Equal(t, "", sortBy, "colon with no field should return empty sortBy")
	require.False(t, sortAsc)
}
