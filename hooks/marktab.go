//
//  Copyright (c) 2024-2025 Chakib Ben Ziane <contact@blob42.xyz>  and [`gosuki` contributors](https://github.com/blob42/gosuki/graphs/contributors).
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

package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/blob42/gosuki"
	"github.com/blob42/gosuki/internal/utils"
	"github.com/blob42/gosuki/pkg/marktab"
	"github.com/blob42/gosuki/pkg/tree"
)

const (
	ArchiveBoxTrigger = "@archivebox"
)

// MarkTab represents a collection of rules defined in the marktab file as lines.
type MarkTab struct {
	Rules []Rule // Rules contains all the parsed rules from the marktab file.
}

type Rule struct {
	Trigger string // keyword to detect in the bookmark tags
	Pattern string // regular expression used for matching against the bookmark URL or title.
	Command string // shell command to execute when both the trigger and pattern match the bookmark tags.

	empty bool // empty is an unexported field indicating whether the rule is empty.
}

// When a rule matches this bookmark, execute the action in the rule.Command
// field. A new shell subprocess is spawned.
//
// The child process receives the following exported fields:
// - $URL
// - $TITLE
// - $TAGS
// - $MODULE
func marktabHook(item any) error {
	err := marktab.PreloadRules()
	if err != nil {
		return err
	}

	switch v := item.(type) {
	case *tree.Node:
		bk := v.GetBookmark()
		if bk == nil {
			panic("unexpected nil bookmark")
		}
		return processMtabHook(bk)
	case *gosuki.Bookmark:
		if v == nil {
			return nil
		}
		return processMtabHook(v)
	default:
		panic("hook: unknown type")
	}
}

func processMtabHook(bk *gosuki.Bookmark) error {
	for _, rule := range marktab.CachedRules.Rules {

		// Spawn a new shell subprocess with the rule's command, passing in
		// the bookmark details.
		if rule.Match(bk) {

			runID := utils.GenStringID(4)

			log.Debug(
				"run marktab",
				"id",
				runID,
				"rule",
				rule.Trigger,
				"url",
				bk.URL,
				"tags",
				strings.Join(bk.Tags, ","),
			)
			cmd := exec.Command("sh", "-c", rule.Command)
			cmd.Env = append(
				os.Environ(),
				"GOSUKI_URL="+bk.URL,
				"GOSUKI_TITLE="+bk.Title,
				"GOSUKI_TAGS="+strings.Join(bk.Tags, ","),
				"GOSUKI_MODULE="+bk.Module,
				"GOSUKI_RUN_ID="+runID,
			)
			if err := cmd.Start(); err != nil {
				return fmt.Errorf("failed to start cmd: %w", err)
			}
			cmd.WaitDelay = time.Second * 60
			if err := cmd.Wait(); err != nil {
				return fmt.Errorf("failed to wait cmd: %w", err)
			}
		}
	}

	return nil
}

func NodeMktabHook(n *tree.Node) error {
	return marktabHook(n)
}

func BkMktabHook(b *gosuki.Bookmark) error {
	return marktabHook(b)
}

func init() {
	registerHook(
		Hook[*tree.Node]{
			name:     "node_marktab",
			Func:     NodeMktabHook,
			priority: 1,
			kind:     GlobalInsertHook | GlobalUpdateHook,
		},
	)
	registerHook(
		Hook[*gosuki.Bookmark]{
			name:     "bk_marktab",
			Func:     BkMktabHook,
			priority: 1,
			kind:     GlobalInsertHook | GlobalUpdateHook,
		},
	)

}
