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

// Implements a channel based inter module communication
package modules

import (
	"context"
	"fmt"
	"time"

	"github.com/blob42/gosuki/pkg/manager"
)

const DispatcherID = "dispatcher"

var (
	MsgDispatcher = modMsgDispatcher{
		listeners: map[ModID]listener{},
	}
)

type ModMsgType string

// types of messages passed between modules
const (
	MsgTriggerSync = "trigger-sync"
	MsgHello       = "mod-hello"
	MsgPanic       = "panic"
)

// channel for sending messages to modules
type ModMsg struct {
	Type ModMsgType
	To   ModID
}

var ModMsgBus = make(chan ModMsg) // Channel for intra-process message passing

// Modules that listen to messages on the inter module channels
type MsgListener interface {
	MsgListen(context.Context, <-chan ModMsg)
}

// Work unit for modules that implement the MsgListener interface
type Listener struct {
	Ctx   context.Context
	Queue chan ModMsg
	MsgListener
}

func (lw Listener) Run(m manager.UnitManager) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				m.Panic(fmt.Errorf("%v", err))
			}
		}()
		lw.MsgListen(lw.Ctx, lw.Queue)
	}()

	<-m.ShouldStop()
	m.Done()
}

type listener struct {
	id    ModID
	queue chan<- ModMsg
}

// dispatchers messages between modules
type modMsgDispatcher struct {
	listeners map[ModID]listener
}

func (mm *modMsgDispatcher) AddListener(id ModID, queue chan<- ModMsg) {
	mm.listeners[id] = listener{id, queue}
}

func (mm *modMsgDispatcher) Run(m manager.UnitManager) {
	go func() {
		log.Info("dispatching module messages")
		for msg := range ModMsgBus {
			log.Debug("dispatching mod message", "msg", msg.Type, "to", msg.To)
			if dst, ok := mm.listeners[msg.To]; ok {
				log.Trace("sending", "msg", msg.Type, "to-mod", msg.To)
				dst.queue <- msg
			} else { // discard
				log.Debugf("target %s not available, discarding msg=%s", msg.To, msg.Type)
			}
		}
	}()

	//DEBUG:
	// time.Sleep(4 * time.Second)
	// ModMsgBus <- ModMsg{MsgHello, "p2p-sync"}

	// Wait for stop signal
	<-m.ShouldStop()
	m.Done()
}

var _ manager.WorkUnit = (*modMsgDispatcher)(nil)
