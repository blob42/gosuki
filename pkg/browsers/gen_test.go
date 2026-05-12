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

package browsers

import (
	"testing"
)

// TestDefinedBrowsersExist ensures that generated browser definitions are available
// This test will fail if go generate has not been run, which would also cause
// builds to fail when installing via `go install`.
func TestDefinedBrowsersExist(t *testing.T) {
	if len(DefinedBrowsers) == 0 {
		t.Fatal("DefinedBrowsers is empty - run 'go generate ./pkg/browsers' to generate browser definitions")
	}

	// Test that Defined function works for each family
	families := []BrowserFamily{Mozilla, ChromeBased, Qutebrowser}
	for _, family := range families {
		defs := Defined(family)
		if family == Mozilla || family == ChromeBased {
			// These families should have at least one browser defined
			if len(defs) == 0 {
				t.Errorf("Defined(%v) returned empty map, expected at least one browser", family)
			}
		}
	}
}
