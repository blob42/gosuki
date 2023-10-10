//
// Copyright ⓒ 2023 Chakib Ben Ziane <contact@blob42.xyz> and [`GoSuki` contributors]
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

package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/blob42/gosuki"
	"github.com/blob42/gosuki/internal/database"
	"github.com/blob42/gosuki/internal/logging"
	"github.com/blob42/gosuki/pkg/manager"
)

type ApiServer struct {
	http.Handler
}

type Bookmark = gosuki.Bookmark

var log = logging.GetLogger("API")

func getBookmarks(w http.ResponseWriter, r *http.Request) {
	rows, err := database.Cache.DB.Handle.QueryContext(r.Context(), "SELECT URL, metadata, tags FROM bookmarks")
	if err != nil {
		log.Error(err)
	}

	var bookmarks []Bookmark
	var tags string

	for rows.Next() {
		bookmark := Bookmark{}
		err = rows.Scan(&bookmark.URL, &bookmark.Metadata, &tags)
		if err != nil {
			log.Error(err)
		}

		bookmark.Tags = strings.Split(tags, database.TagSep)
		//log.Debug("GET %s", tags)
		//log.Debug("%v", bookmark)

		bookmarks = append(bookmarks, bookmark)
	}

	err = json.NewEncoder(w).Encode(bookmarks)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Run router
// TODO: config params for api
func (s *ApiServer) Run(m manager.UnitManager) {
	server := &http.Server{
		Addr:         ":4444",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 90 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      s.Handler,
	}
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			if err != http.ErrServerClosed {
				m.Panic(err)
			}
		}
	}()

	// Wait for stop signal
	<-m.ShouldStop()
	m.Done()
}

func NewApi() *ApiServer {

	router := chi.NewRouter()
	router.Use(middleware.Logger)

	router.HandleFunc("/bookmarks", getBookmarks)

	return &ApiServer{router}
}
