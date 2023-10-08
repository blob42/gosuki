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

package parsing

import (
	"time"

	"github.com/blob42/gosuki/internal/logging"
	"github.com/blob42/gosuki/pkg/tree"
)

type Node = tree.Node

var log = logging.GetLogger("PARSE")

const (

)

type Stats struct {
	LastFullTreeParseTime time.Duration
	LastWatchRunTime      time.Duration
	LastNodeCount         int
	LastURLCount          int
	CurrentNodeCount      int
	CurrentURLCount       int
}

func (s *Stats) Reset() {
    s.LastURLCount = s.CurrentURLCount
	s.LastNodeCount = s.CurrentNodeCount
	s.CurrentNodeCount = 0
	s.CurrentURLCount = 0
}


