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

const (
	CacheName   = "memcache"
	L2CacheName = "memcache_l2"
	//MemcacheFmt = "file:%s?mode=memory&cache=shared"
	//BufferFmt   = "file:%s?mode=memory&cache=shared"
	DBTypeInMemoryDSN = "file:%s?mode=memory&cache=shared&_journal=MEMORY"
	DBTypeCacheDSN    = DBTypeInMemoryDSN
)

// Global in-memory cache hierarchy for the gosuki database, structured in two
// levels to optimize performance and reduce unnecessary I/O operations.
//
// Cache (level 1 cache) serves as a working buffer that aggregates and merges
// data from all scanned bookmarks. It acts as the primary cache for real-time
// operations and is periodically synchronized with L2Cache.
//
// L2Cache (level 2 cache) functions as a persistent memory mirror of the on-disk
// database (gosuki.db). It is updated as a final step after level 1 cache
// synchronizations from module buffers, enabling checksum-based comparison
// between levels to detect changes and avoid redundant updates. This ensures
// efficient data consistency checks and minimizes write operations to the
// underlying storage.
//
// The two-level architecture balances speed (level 1) with data integrity (level 2),
// while L2Cache maintains a faithful in-memory replica of the on-disk database state.
var (
	Cache   = &CacheDB{}
	L2Cache = &CacheDB{}
)

type CacheDB struct {
	*DB
}

func GetCacheDB() *CacheDB {
	if !Cache.IsInitialized() {
		log.Fatal("cache is not initialized")
	}
	return Cache
}

func (c *CacheDB) IsInitialized() bool {
	return Cache.DB != nil && Cache.Handle != nil
}

func initCache() {
	log.Debug("initializing cacheDB")
	var err error

	// Initialize memory db with schema
	Cache.DB, err = NewDB(CacheName, "", DBTypeCacheDSN).Init()
	if err != nil {
		log.Fatal(err)
	}

	err = Cache.InitSchema()
	if err != nil {
		log.Fatal(err)
	}

	L2Cache.DB, err = NewDB(L2CacheName, "", DBTypeCacheDSN).Init()
	if err != nil {
		log.Fatal(err)
	}
	err = L2Cache.InitSchema()
	if err != nil {
		log.Fatal(err)
	}

	//TEST: sqlite table locked
	// Cache.Handle.SetMaxIdleConns(1)
}
