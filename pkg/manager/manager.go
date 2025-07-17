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

package manager

import (
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"sync"

	"slices"

	"github.com/blob42/gosuki/internal/webui"
	"github.com/blob42/gosuki/pkg/logging"
)

var (
	idGenerator = genID()
	log         = logging.GetLogger("MNGR")
)

// The WorkUnit interface is used to define a unit of work.
// The Run method will be called in a goroutine.
type WorkUnit interface {
	Run(UnitManager)
}

// The UnitManager interface is used to manage a unit of work.
// The ShouldStop method returns a channel that will be closed when the unit
// should stop.
// The Done method should be called when the unit is done.
type UnitManager interface {
	ShouldStop() <-chan bool
	Done()
	Panic(err error)

	RequestShutdown()
}

type WorkUnitManager struct {
	name       string
	stop       chan bool
	workerQuit chan bool
	unit       WorkUnit
	panic      chan error
	isPaniced  bool
}

func (w *WorkUnitManager) ShouldStop() <-chan bool {
	return w.stop
}

func (w *WorkUnitManager) Done() {
	w.workerQuit <- true
}

func (w *WorkUnitManager) Panic(err error) {
	w.panic <- err
	w.isPaniced = true
	w.workerQuit <- true
}

func (m *WorkUnitManager) RequestShutdown() {
	m.panic <- fmt.Errorf("request for shutdown")
}

type Manager struct {
	signalIn     chan os.Signal              // Channel for incoming system signals
	shutdownSigs []os.Signal                 // List of accepted OS shutdown signals
	workers      map[string]*WorkUnitManager // Map of worker units managed by the manager
	Quit         chan bool                   // Channel to signal that the manager is done
	ready        chan bool                   // Signal channel indicating all workers are running
	panic        chan error                  // Channel for panicking goroutines, used to force shutdown
	mu           sync.Mutex
}

func (m *Manager) Shutdown() {
	<-m.ready
	// send shutdown event to all worker units
	for name, w := range m.workers {
		log.Debugf("stopping %s\n", name)
		w.stop <- true
	}

	// Wait for all units to quit
	for name, w := range m.workers {
		<-w.workerQuit
		log.Debugf("%s down", name)
	}

	// All workers have shutdown
	log.Info("all workers down, stopping manager ...")

	m.Quit <- true
}

func (m *Manager) Start() {
	log.Debug("starting manager ...")

	// for unitName, w := range m.workers {
	// 	log.Info("starting", "unit", unitName)
	// 	go w.unit.Run(w)
	// }

	m.ready <- true

	log.Info("manager is up")
	if !logging.TUIMode {
		fmt.Println("gosuki service up and running")
		for name := range m.Units() {
			if strings.HasPrefix(name, "webui") {
				fmt.Printf("WebUI listening on: http://%s\n", webui.BindAddr)
			}
		}
	}

	for {
		select {
		case sig := <-m.signalIn:
			// log.Debugf("%#v\n", sig)

			if !in(m.shutdownSigs, sig) {
				break
			}

			log.Debug("quit event received ... ")
			m.Shutdown()

		case p := <-m.panic:

			for name, w := range m.workers {
				if w.isPaniced {
					log.Errorf("Panicing for <%s>: %s", name, p)
				} else {
					log.Debugf("shuting down <%s>\n", name)
					w.stop <- true
					<-w.workerQuit
					log.Debugf("<%s> down", name)
				}
			}

			// All workers have shutdown
			log.Info("All workers shutdown, shutting down manager ...")

			m.Quit <- true

		}
	}
}

func (m *Manager) ShutdownOn(sig ...os.Signal) {

	log.Debugf("Registering shutdown signals: %v", sig)
	signal.Notify(m.signalIn, sig...)

	m.shutdownSigs = append(m.shutdownSigs, sig...)
}

type IDGenerator func(string) int

func genID() IDGenerator {
	ids := make(map[string]int)

	return func(unit string) int {
		ret := ids[unit]
		ids[unit]++
		return ret
	}
}

func (m *Manager) AddUnit(unit WorkUnit, name string) {

	workUnitManager := &WorkUnitManager{
		name:       name,
		workerQuit: make(chan bool, 1),
		stop:       make(chan bool, 1),
		unit:       unit,
		panic:      m.panic,
	}

	unitType := reflect.TypeOf(unit)
	unitClass := strings.Split(unitType.String(), ".")[1]
	unitName := fmt.Sprintf("%s[%s", name, unitClass)
	unitID := idGenerator(unitName)
	unitName = fmt.Sprintf("%s#%d]", unitName, unitID)

	log.Debug("adding unit ", "name", unitName)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.workers[unitName] = workUnitManager

	// Launch the unit's goroutine *immediatly*
	go func() {
		defer func() {
			// Handle panics within the unit's goroutine
			if r := recover(); r != nil {
				m.panic <- fmt.Errorf("unit %s panicked: %v", unitName, r)
				workUnitManager.isPaniced = true
			}
		}()
		log.Info("starting", "unit", unitName)
		workUnitManager.unit.Run(workUnitManager)
	}()

}

func NewManager() *Manager {
	return &Manager{
		signalIn: make(chan os.Signal, 1),
		Quit:     make(chan bool, 1),
		workers:  make(map[string]*WorkUnitManager),
		panic:    make(chan error, 1),
		ready:    make(chan bool, 1),
	}
}

func (m *Manager) Units() map[string]WorkUnit {
	res := map[string]WorkUnit{}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, workUnit := range m.workers {
		res[workUnit.name] = workUnit.unit
	}
	return res
}

// Test if signal is in array
func in(arr []os.Signal, sig os.Signal) bool {
	return slices.Contains(arr, sig)
}
