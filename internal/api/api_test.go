package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetPaginationParams_SortBasic(t *testing.T) {
	tests := []struct {
		name    string
		sort    string
		sortBy  string
		sortAsc bool
	}{
		{"modified desc", "modified", "modified", false},
		{"modified asc", "modified:asc", "modified", true},
		{"modified DESC", "modified:DESC", "modified", false},
		{"title asc", "title:asc", "title", true},
		{"title desc", "title:desc", "title", false},
		{"url asc", "url:asc", "url", true},
		{"url desc", "url:desc", "url", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encoded := url.QueryEscape(tc.sort)
			r := httptest.NewRequest(http.MethodGet, "/?sort="+encoded, nil)
			params := GetPaginationParams(r)
			require.Equal(t, tc.sortBy, params.SortBy)
			require.Equal(t, tc.sortAsc, params.SortAsc)
		})
	}
}

func TestGetPaginationParams_SortWhitespace(t *testing.T) {
	encoded := url.QueryEscape("modified : asc")
	r := httptest.NewRequest(http.MethodGet, "/?sort="+encoded, nil)
	params := GetPaginationParams(r)
	require.Equal(t, "modified", params.SortBy)
	require.True(t, params.SortAsc)
}

func TestGetPaginationParams_SortEmpty(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?sort=", nil)
	params := GetPaginationParams(r)
	require.Equal(t, "", params.SortBy)
}

func TestGetPaginationParams_SortWhitespaceOnly(t *testing.T) {
	encoded := url.QueryEscape("   ")
	r := httptest.NewRequest(http.MethodGet, "/?sort="+encoded, nil)
	params := GetPaginationParams(r)
	require.Equal(t, "", params.SortBy)
}

func TestGetPaginationParams_SortCombinedWithPage(t *testing.T) {
	encoded := url.QueryEscape("title:desc")
	r := httptest.NewRequest(http.MethodGet, "/?page=2&per_page=10&sort="+encoded, nil)
	params := GetPaginationParams(r)

	require.Equal(t, 2, params.Page)
	require.Equal(t, 10, params.Size)
	require.Equal(t, "title", params.SortBy)
	require.False(t, params.SortAsc)
}

func TestGetPaginationParams_NoSortParam(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?page=1", nil)
	params := GetPaginationParams(r)
	require.Equal(t, "", params.SortBy)
	require.False(t, params.SortAsc)
	require.Equal(t, 1, params.Page)
}
