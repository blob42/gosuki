package main

import (
	"fmt"
	"os"

	"git.blob42.xyz/gomark/gosuki/modules"
	"git.blob42.xyz/gomark/gosuki/parsing"
	"git.blob42.xyz/gomark/gosuki/profiles"
	"git.blob42.xyz/gomark/gosuki/utils"
	"git.blob42.xyz/gomark/gosuki/watch"

	"git.blob42.xyz/sp4ke/gum"

	"github.com/urfave/cli/v2"
)

var startDaemonCmd = &cli.Command{
	Name:    "daemon",
	Aliases: []string{"d"},
	Usage:   "run browser watchers",
	// Category: "daemon"
	Action:  startDaemon,
}

// Runs the module by calling the setup 
func RunModule(c *cli.Context,
				browserMod modules.BrowserModule,
				p *profiles.Profile) error {
		mod := browserMod.ModInfo()
		// Create context
		modContext := &modules.Context{
			Cli: c,
		}
		//Create a browser instance
		browser, ok := mod.New().(modules.BrowserModule)
		if !ok {
			log.Criticalf("module <%s> is not a BrowserModule", mod.ID)
		}
		log.Debugf("created browser instance <%s>", browser.Config().Name)

		// shutdown logic
		s, isShutdowner := browser.(modules.Shutdowner)
		if isShutdowner {
			defer handleShutdown(browser.Config().Name, s)
		}

		log.Debugf("new browser <%s> instance", browser.Config().Name)
		h, ok := browser.(modules.HookRunner)
		if ok {
			//TODO: document hook running on watch events
			h.RegisterHooks(parsing.ParseTags)
		}


		// calls the setup logic for each browser instance which
		// includes the browsers.Initializer and browsers.Loader interfaces

		//TODO!: call with custom profile
		if p != nil {
			bpm, ok := browser.(profiles.ProfileManager)
			if !ok {
				err := fmt.Errorf("<%s> does not implement profiles.ProfileManager",
				browser.Config().Name)
				log.Critical(err)
				return err
			}
			if err := bpm.UseProfile(*p); err != nil {
				log.Criticalf("could not use profile <%s>", p.Name)
				return err
			}
		}


		err := modules.Setup(browser, modContext)
		if err != nil {
			log.Errorf("setting up <%s> %v", browser.Config().Name, err)
			if isShutdowner {
				handleShutdown(browser.Config().Name, s)
			}
			return err
		}

		runner, ok := browser.(watch.WatchRunner)
		if !ok {
			err =  fmt.Errorf("<%s> must implement watch.WatchRunner interface", browser.Config().Name)
			log.Critical(err)
			return err
		}

		log.Infof("start watching <%s>", runner.Watch().ID)
		watch.SpawnWatcher(runner)
		return nil
}

func startDaemon(c *cli.Context) error {
	defer utils.CleanFiles()
	manager := gum.NewManager()
	manager.ShutdownOn(os.Interrupt)

	api := NewApi()
	manager.AddUnit(api)

	go manager.Run()

	// Initialize sqlite database available in global `cacheDB` variable
	initDB()

	registeredBrowsers := modules.GetBrowserModules()

	// instanciate all browsers
	for _, browserMod := range registeredBrowsers {

		mod := browserMod.ModInfo()

		//Create a temporary browser instance to check if it implements
		// the ProfileManager interface
		browser, ok := mod.New().(modules.BrowserModule)
		if !ok {
			log.Criticalf("module <%s> is not a BrowserModule", mod.ID)
		}

		//WIP: Handle multiple profiles for modules who announce it - here ?
		// Check if browser implements ProfileManager
		//WIP: global flag for watch all

		// Check if watch all profiles is defined
		// if defined then spawn a new browser module for each profile
		bpm, ok := browser.(profiles.ProfileManager)
		if ok {
			if bpm.WatchAllProfiles() {
				profs, err := bpm.GetProfiles()
				if err != nil {
					log.Critical("could not get profiles")
					continue
				}
				for _, p := range profs {
					log.Debugf("profile: <%s>", p.Name)
					err = RunModule(c, browserMod, p)
					if err != nil {
					  continue
					}
				}
			} else {
				err := RunModule(c, browserMod, nil)
				if err != nil {
				  continue
				}
			}
		} else {
			log.Warningf("module <%s> does not implement profiles.ProfileManager",
			browser.Config().Name)
		}
	}

	<-manager.Quit

	return nil
}

func handleShutdown(id string, s modules.Shutdowner) {
	err := s.Shutdown()
	if err != nil {
		log.Panicf("could not shutdown browser <%s>", id)
	}
}
