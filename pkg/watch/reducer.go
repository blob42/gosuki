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

package watch

import "time"

// Run reducer in its own thread when the watcher is started
// It receives a struct{event, func} and runs the func only once in the interval
func ReduceEvents(interval time.Duration,
	w WatchRunner) {
	watch := w.Watch()
	log.Debugf("starting reducer service for %s", watch.ID)

	eventsIn := w.Watch().eventsChan
	timer := time.NewTimer(interval)
	beat := time.NewTicker(1 * time.Second).C
	var events []bool

	for {
		select {
		case <-eventsIn:
			log.Trace("[reducuer] received event, resetting watch interval")
			timer.Reset(interval)
			events = append(events, true)

		case <-beat:
			// log.Trace("reducer", "beat", watch.ID)

		case <-timer.C:
			if len(events) > 0 {
				log.Debug("<reduce>: calling Run()")
				w.Run()

				// Empty events queue
				events = make([]bool, 0)
			}
		case _, ok := <-eventsIn:
			if !ok {
				log.Warnf("Events channel closed for %s", watch.ID)
				return // Exit the function or handle the closed channel case
			}
		}
	}
}
