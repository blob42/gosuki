// Copyright (c) 2023 Chakib Ben Ziane <contact@blob42.xyz>  and [`gosuki` contributors](https://github.com/blob42/gosuki/graphs/contributors).
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
package chrome

import (
	"fmt"
	"testing"

	"github.com/blob42/gosuki/internal/utils"
)

const statePath = "~/.config/google-chrome/Local State"

func TestLoadLocalState(t *testing.T) {
	var state *StateData
	fullPath, err := utils.ExpandPath(statePath)
	if err != nil {
		t.Fatal(err)
	}
	state, err = loadLocalState(fullPath)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("\nstate:\n%+v\n", state)
}

func TestGetProfiles(t *testing.T) {
	ch := &Chrome{}
 	profiles, err := ch.GetProfiles(ChromeStable)
	if err != nil {
		t.Fatal(err)
	}
	for _, prof := range profiles {
		fmt.Printf("\nprofiles:\n%#v\n", prof)
		fmt.Println(prof.AbsolutePath())
	}
}
