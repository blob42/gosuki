// ### API Usage:
// - Init Mode (debug, release) and Logging
// - Create a Browser object for each browser using `New[BrowserType]()`
// - Call `Load()` and `Watch()` on every browser
// - Run the gin server
package main

import (
	"os"

	"git.sp4ke.xyz/sp4ke/gomark/browsers"
	"git.sp4ke.xyz/sp4ke/gomark/config"

	"git.sp4ke.xyz/sp4ke/gomark/cmd"

	"github.com/urfave/cli/v2"
	// Load firefox browser modules
	_ "git.sp4ke.xyz/sp4ke/gomark/firefox"
)

func main() {
	app := cli.NewApp()
	app.Name = "gomark"
	app.Version = version()

	flags := []cli.Flag{

		&cli.StringFlag{
			Name:  "config-file",
			Value: config.ConfigFile,
			Usage: "TOML config `FILE` path",
		},
	}

	app.Before = func(c *cli.Context) error {

		// get all registered browser modules
		modules := browsers.Modules()
		for _, mod := range modules {

			// Run module's before context hooks
			// for example setup flags management
			err := cmd.BeforeHook(string(mod.ModInfo().ID))(c)
			if err != nil {
				return err
			}
		}

		// Execute config hooks
		//TODO: better doc for what are Conf hooks ???
		config.RunConfHooks()

		return nil
	}

	app.Flags = flags

	// Browser modules can register commands through cmd.RegisterModCommand.
	// registered commands will be appended here
	app.Commands = []*cli.Command{
		startServerCmd,
		cmd.ConfigCmds,
	}

	// Add global flags from registered modules
	modules := browsers.Modules()
	for _, mod := range modules {
		modId := string(mod.ModInfo().ID)

		// for each registered module, register own flag management
		mod_flags := cmd.GlobalFlags(modId)
		if len(mod_flags) != 0 {
			app.Flags = append(app.Flags, mod_flags...)
		}

		// Add all browser module registered commands
		cmds := cmd.ModCommands(modId)
		for _, cmd := range cmds {
			app.Commands = append(app.Commands, cmd)
		}
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func init() {
	//TODO: register global flags as cli flags
	config.RegisterGlobalOption("myglobal", 1)

	// First load or bootstrap config
	//TEST: load order of init
	initConfig()
}
