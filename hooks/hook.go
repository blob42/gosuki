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

// Package hooks permits to register custom hooks that will be called during the parsing
// process of a bookmark file. Hooks can be used to extract tags, commands or other custom
// data from a bookmark title or description.
//
// They can effectively be used as a command line interface to the host system
// through the browser builtin Ctrl+D bookmark feature.
package hooks

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/blob42/gosuki"
	"github.com/blob42/gosuki/pkg/logging"
	"github.com/blob42/gosuki/pkg/tree"
)

var (
	log = logging.GetLogger("hook")
)

// Hookable defines types that can have hooks applied to them, either
// *gosuki.Bookmark or *tree.Node.
type Hookable interface {
	*gosuki.Bookmark | *tree.Node
}

type HookJob struct {
	Book *gosuki.Bookmark
	Kind Kind
}

// Kind represents the category of a hook, determining when and how it is
// executed.
type Kind int

const (
	// BrowserHook is executed for each bookmark loading in every browser
	// instance. Primarily used for internal gosuki operations, this hook runs
	// on all bookmarks whenever a bookmark change occurs.
	BrowserHook = 1 << iota

	//  triggered when bookmarks are inserted the main database
	GlobalInsertHook

	// triggered when bookmarks are updated in the main database
	GlobalUpdateHook
)

// A Hook is a function that takes a Hookable type (*Bookmark or *Node) and
// performs an arbitrary process. Hooks are executed during bookmark loading or
// real-time detection of changes.
//
// For example, the TAG extraction process is managed by the ParseXTags hook.
//
// Hooks can also be used to handle custom user commands and messages found in
// bookmark fields.
type Hook[T Hookable] struct {
	// Unique name of the hook, used for identification and debugging.
	name string

	// Function to call on a node or bookmark. Must return an error if
	// processing fails.
	Func func(T) error

	// Priority determines the execution order of hooks. Higher priority (lower
	// value) runs first.
	priority uint

	// Kind specifies the category of the hook, determining its execution
	// context.
	kind Kind
}

func (h Hook[T]) Name() string {
	return h.name
}

func (h Hook[T]) Kind() Kind {
	return h.kind
}

// SortByPriority sorts a slice of NamedHook by priority, with higher priority
// (lower uint value) first. This uses reflection to access the priority field
// of each hook.
func SortByPriority(hooks []NamedHook) {
	sort.Slice(hooks, func(i, j int) bool {
		vi := reflect.ValueOf(hooks[i])
		vj := reflect.ValueOf(hooks[j])

		if vi.Kind() != reflect.Struct || vj.Kind() != reflect.Struct {
			panic("expected struct")
		}

		pi := vi.FieldByName("priority")
		pj := vj.FieldByName("priority")

		if !pi.IsValid() || !pj.IsValid() {
			panic("missing priority field")
		}

		return pi.Uint() < pj.Uint()
	})
}

// HookRunner defines the interface for browsers that can register custom hooks.
// Implementers can define hooks that are executed during the main Run() method
// to process commands and messages found in tags or parsed data from browsers.
type HookRunner interface {

	// CallHooks executes all registered hooks on the provided target (e.g., a
	// node or bookmark). This method is typically called during the main Run()
	// lifecycle of a browser.
	CallHooks(any) error
}

func processGlobalHooks(hj HookJob) error {
	for name, hook := range Defined {
		if bkHook, ok := hook.(Hook[*gosuki.Bookmark]); ok {
			if (hook.Kind() & hj.Kind) != 0 {
				if err := bkHook.Func(hj.Book); err != nil {
					return fmt.Errorf("hook %s error :%w", name, err)
				}

			}

		}
	}
	return nil
}

// HooksScheduler calls bookmark hooks on queued hook jobs
func HooksScheduler(incoming <-chan HookJob) {
	hookErrors := make(chan error, 10)
	defer close(hookErrors)

	processHook := func(hj HookJob) {
		if err := processGlobalHooks(hj); err != nil {
			select {
			case hookErrors <- err:
			default:
				log.Error("Dropped hook error: ", err)
			}
		}
	}

	for {
		select {
		case book, ok := <-incoming:
			if !ok {
				return
			}
			// Execute hooks sequentially to prevent race conditions in user scripts
			processHook(book)
		case err := <-hookErrors:
			log.Error(err)

		}
	}
}
