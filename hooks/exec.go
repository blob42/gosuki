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

//go:build linux

package hooks

// Hooks to exececute host system commands

import (
	"regexp"

	"github.com/0xAX/notificator"

	"github.com/blob42/gosuki/pkg/parsing"
	"github.com/blob42/gosuki/pkg/tree"
)

// Hook that sends a system notification using notify-send (Linux).
// To enable notification a tag must be presetn in the name with the form: `#tag:notify`
// Requires notify-send to be installed
func NotifySend(node *tree.Node) error {
	// if node.Name has a tag of the form #tag:notify

	regex := regexp.MustCompile(parsing.ReNotify)
	
	if !regex.MatchString(node.Name) {
		return nil
	}

	notify := notificator.New(notificator.Options{
		AppName: "gosuki",
	})
	return notify.Push("new bookmark", node.URL, "", notificator.UR_NORMAL)
}

func init(){
	regHook(Hook{
		Name: "notify-send",
		Func: NotifySend,
	})
}




