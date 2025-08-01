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

package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"

	db "github.com/blob42/gosuki/internal/database"
	"github.com/blob42/gosuki/pkg/browsers"
	"github.com/blob42/gosuki/pkg/config"
	"github.com/blob42/gosuki/pkg/events"
	"github.com/blob42/gosuki/pkg/modules"
	"github.com/blob42/gosuki/pkg/profiles"
	"github.com/blob42/gosuki/pkg/watch"

	"github.com/blob42/gosuki/pkg/manager"

	"github.com/urfave/cli/v3"
)

var startDaemonCmd = &cli.Command{
	Name:    "start",
	Aliases: []string{"s"},
	Usage:   "Starts the gosuki service",
	// Category: "daemon"
	Action: startDaemon,
}

// Runs the module by calling the setup
func runBrowserModule(m *manager.Manager,
	ctx context.Context,
	cmd *cli.Command,
	browserMod modules.BrowserModule,
	pfl *profiles.Profile,
	flav *browsers.BrowserDef) error {
	var profileName string
	mod := browserMod.ModInfo()

	// context for module
	modContext := &modules.Context{
		Context: ctx,
		Cli:     cmd,
		IsTUI:   cmd.Bool("tui") && isatty.IsTerminal(os.Stdout.Fd()),
	}

	//Create a browser instance
	browser, ok := mod.New().(modules.BrowserModule)
	if !ok {
		return fmt.Errorf("module <%s> is not a BrowserModule", mod.ID)
	}
	config := browser.Config()
	log.Debugf("created browser instance <%s>", config.Name)

	// shutdown logic
	_, isShutdowner := browser.(modules.Shutdowner)
	if !isShutdowner {
		log.Warn("does not implement modules.Shutdowner", "browser", config.Name)
	}

	if pfl != nil {
		bpm, ok := browser.(profiles.ProfileManager)
		if !ok {
			err := fmt.Errorf("<%s> does not implement profiles.ProfileManager",
				config.Name)
			log.Error(err)
			return err
		}
		if err := bpm.UseProfile(pfl, flav); err != nil {
			log.Warnf("unable to load profile <%s.%s>: %s", mod.ID, pfl.Name, err)
			return &modules.ErrModDisabled{Err: err}
		}
		profileName = pfl.Name
	}

	runner, ok := browser.(watch.WatchRunner)
	if !ok {
		return errors.New("must implement watch.WatchRunner interface")
	}

	go func() {
		events.TUIBus <- events.RunnerStarted{WatchRunner: runner}
	}()

	// calls the setup logic for each browser instance which
	// includes the browsers.Initializer and browsers.Loader interfaces
	//PERF:
	if err := modules.SetupBrowser(browser, modContext, pfl); err != nil {
		return err
	}

	w := runner.Watch()
	if w == nil {
		return errors.New("must return a valid watch descriptor")
	}
	log.Debugf("adding watch runner <%s>", runner.Watch().ID)

	// create the worker name
	unitName := config.Name
	if len(profileName) > 0 {
		unitName = fmt.Sprintf("%s(%s)", unitName, profileName)
	}

	worker := watch.WatchWork{
		WatchRunner: runner,
	}

	m.AddUnit(worker, unitName)

	return nil
}

func startNormalDaemon(ctx context.Context, cmd *cli.Command, mngr *manager.Manager) error {
	defer func(m *manager.Manager) {
		go m.Start()
	}(mngr)

	// Initialize sqlite database available in global `cacheDB` variable
	db.Init()

	// Handle generic modules
	mods := modules.GetModules()
	for _, mod := range mods {
		name := mod.ModInfo().ID
		modInstance := mod.ModInfo().New()
		if _, ok := modInstance.(modules.BrowserModule); ok {
			log.Debugf("skipping non browser module")
			continue
		}

		log.Debugf("starting <%s>", name)

		modContext := &modules.Context{
			Context: ctx,
			Cli:     cmd,
			IsTUI:   cmd.Bool("tui") && isatty.IsTerminal(os.Stdout.Fd()),
		}

		// A generic modules need to implement on of:
		// - watch.Poller
		// - watch.WatchLoader
		// - and (optionally) modules.MsgListener
		// OR
		// - ONLY implement modules.MsgListener
		var worker manager.WorkUnit
		listener, isMsgListener := modInstance.(modules.MsgListener)
		listenerQueue := make(chan modules.ModMsg, 64)

		// Check for Poller or Loader first
		if poller, ok := modInstance.(watch.Poller); ok {
			worker = watch.PollWork{
				Name:   string(name),
				Poller: poller,
			}

			// Check if it's also a MsgListener
			if isMsgListener {
				listeningWorker := modules.Listener{
					Queue:       listenerQueue,
					MsgListener: listener,
				}
				mngr.AddUnit(listeningWorker, string(name))
			}
		} else if loader, ok := modInstance.(watch.WatchLoader); ok {
			worker = watch.WatchLoad{
				WatchLoader: loader,
			}

			// Check if it's also a MsgListener
			if isMsgListener {
				listeningWorker := modules.Listener{
					Queue:       listenerQueue,
					MsgListener: listener,
				}
				mngr.AddUnit(listeningWorker, string(name))
			}
		} else if isMsgListener {
			worker = modules.Listener{
				Queue:       listenerQueue,
				MsgListener: listener,
			}
		}

		// Setup the module
		if err := modules.SetupModule(mod, modContext); err != nil {
			log.Warn(err, "mod", name)
			continue
		}

		mngr.AddUnit(worker, string(name))

		// Register as a message listener if applicable
		if isMsgListener {
			modules.MsgDispatcher.AddListener(name, listenerQueue)
		}
	}

	// start all registered browser modules
	registeredBrowsers := modules.GetBrowserModules()

	// instanciate all browsers
	for _, browserMod := range registeredBrowsers {
		mod := browserMod.ModInfo()
		log.Infof("starting <%s>", mod.ID)

		//Create a temporary browser instance to check if it implements
		// the ProfileManager interface
		browser, ok := mod.New().(modules.BrowserModule)
		if !ok {
			log.Fatalf("Module <%s> is not a BrowserModule", mod.ID)
		}

		// call runModule for each profile
		bpm, ok := browser.(profiles.ProfileManager)
		if ok {
			if cmd.Bool("watch-all") ||
				(config.GlobalConfig.WatchAll ||
					bpm.WatchAllProfiles()) {
				flavours := bpm.ListFlavours()
				for _, flav := range flavours {
					profs, err := bpm.GetProfiles(flav.Flavour)
					if err != nil {
						log.Info("no profiles found", "browser", flav.Flavour)
						continue
					}
					for _, p := range profs {
						log.Debug("", "flavour", flav.Flavour, "profile", p.Name)
						err = runBrowserModule(mngr, ctx, cmd, browserMod, p, &flav)
						if err != nil {
							if errDisabled, errDisable := err.(*modules.ErrModDisabled); errDisable {
								log.Warn("disabling browser profile", "profile", p.Name, "mod", browserMod.ModInfo().ID, "reason", errDisabled.Reason)
								modules.Disable(browserMod.ModInfo().ID)
							} else {
								log.Error(err, "browser", flav.Flavour)
							}
							continue
						}
					}
				}
			} else {
				log.Debugf("profile manager <%s> not watching all profiles",
					browser.Config().Name)
				err := runBrowserModule(mngr, ctx, cmd, browserMod, nil, nil)
				if err != nil {
					if _, errDisable := err.(*modules.ErrModDisabled); errDisable {
						log.Warn("disabling browser", "mod", browserMod.ModInfo().ID)
						modules.Disable(browserMod.ModInfo().ID)
					} else {
						log.Error(err, "browser", browserMod.Config().Name)
					}
					continue
				}
			}
		} else {
			log.Info("not implemented profiles.ProfileManager", "browser",
				browser.Config().Name)
			if err := runBrowserModule(mngr, ctx, cmd, browserMod, nil, nil); err != nil {
				if _, errDisable := err.(*modules.ErrModDisabled); errDisable {
					log.Warn("disabling browser", "mod", browserMod.ModInfo().ID)
					modules.Disable(browserMod.ModInfo().ID)
				} else {
					log.Error(err, "browser", browser.Config().Name)
				}
				continue
			}
		}
	}

	return nil
}
