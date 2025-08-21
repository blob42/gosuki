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

//go:build linux

// This is an example implementation of a hook that simply sends a notification using the
// notification daemon of the system.
package hooks

import (
	"regexp"
	"slices"

	"github.com/0xAX/notificator"

	"github.com/blob42/gosuki"
	"github.com/blob42/gosuki/pkg/parsing"
	"github.com/blob42/gosuki/pkg/tree"
)

// DEBUG:
// Hook that sends a system notification using notify-send (Linux).
// To enable notification a tag must be presetn in the name with the form: `#tag:notify`
// Requires notify-send to be installed
// if node.Name has a tag of the form #tag:notify
func notifySend(item any) error {
	var url string
	regex := regexp.MustCompile(parsing.ReNotify)

	switch v := item.(type) {
	case *tree.Node:
		if !regex.MatchString(v.Title) && !slices.ContainsFunc(v.Tags, func(t string) bool {
			return regex.MatchString(t)
		}) {
			return nil
		}
		url = v.URL

	case *gosuki.Bookmark:

		// if mytag:notify is in title or any of the bookmark tags
		if !regex.MatchString(v.Title) && !slices.ContainsFunc(v.Tags, func(t string) bool {
			return regex.MatchString(t)
		}) {
			return nil
		}
		url = v.URL
	default:
		panic("hook: unknown type")
	}

	notify := notificator.New(notificator.Options{
		AppName: "gosuki",
	})
	return notify.Push("new bookmark", url, "", notificator.UR_NORMAL)
}

func NodeNotifySend(n *tree.Node) error {
	return notifySend(n)
}

func BkNotifySend(b *gosuki.Bookmark) error {
	return notifySend(b)
}

func init() {
	registerHook(
		Hook[*tree.Node]{
			name:     "node_notify_send",
			Func:     NodeNotifySend,
			priority: 20,
		})
	registerHook(
		Hook[*gosuki.Bookmark]{
			name:     "bk_notify_send",
			Func:     BkNotifySend,
			priority: 20,
		})
}
