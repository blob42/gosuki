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

// TODO: save config back to file
// TODO: global config options should be automatically shown in cli global flags
package config

import (
	"fmt"

	"github.com/blob42/gosuki/internal/logging"

	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"
	"github.com/urfave/cli/v2"
)

type Hook func(c *cli.Context) error

var (
	log            = logging.GetLogger("CONF")
	ConfReadyHooks []Hook
	configs        = make(map[string]Configurator)
)

const (
	GlobalConfigName = "global"
)

// A Configurator allows multiple packages and modules to set and access configs
// which can be mapped to any output format (toml, cli flags, env variables ...)
type Configurator interface {
	Set(opt string, v interface{}) error
	Get(opt string) (interface{}, error)
	Dump() map[string]interface{}
	MapFrom(interface{}) error
}

// Used to store the global config
type Config map[string]interface{}

func (c Config) Set(opt string, v interface{}) error {
	c[opt] = v
	return nil
}

func (c Config) Get(opt string) (interface{}, error) {
	return c[opt], nil
}

func (c Config) Dump() map[string]interface{} {
	return c
}

func (c Config) MapFrom(src interface{}) error {
	// Not used here
	return nil
}

type AutoConfigurator struct{
	c interface{}
}

func (ac AutoConfigurator) Set(opt string, v interface{}) error {
	// log.Debugf("setting option %s = %v", opt, v)
	s := structs.New(ac.c)
	f, ok := s.FieldOk(opt)
	if !ok {
		return fmt.Errorf("%s option not defined", opt)
	}

	return f.Set(v)
}

func (ac AutoConfigurator) Get(opt string) (interface{}, error) {
	s := structs.New(ac.c)
	f, ok := s.FieldOk(opt)
	if !ok {
		return nil, fmt.Errorf("%s option not defined", opt)
	}

	return f.Value(), nil
}

func (ac AutoConfigurator) Dump() map[string]interface{} {
	s := structs.New(ac.c)
	return s.Map()
}

func (ac AutoConfigurator) MapFrom(src interface{}) error {
	log.Debugf("mapping from:  %#v ", src)
	log.Debugf("mapping to:  %#v ", ac.c)
	return mapstructure.Decode(src, ac.c)
}

func AsConfigurator(c interface{}) Configurator {
 	return AutoConfigurator{c}
}

// Register a global option ie. under [global] in toml file
func RegisterGlobalOption(key string, val interface{}) {
	log.Debugf("Registring global option %s = %v", key, val)
	configs[GlobalConfigName].Set(key, val)
}

func RegisterModuleOpt(module string, opt string, val interface{}) error {
	log.Debugf("Setting option for module <%s>: %s = %v", module, opt, val)
	dest := configs[module]
	if err := dest.Set(opt, val); err != nil {
		return err
	}
	watchAll, _ := configs[module].Get("WatchAllProfiles")
	log.Debugf("[%s]WATCH_ALL: %v", module, watchAll)
	return nil
}

// Get all configs as a map[string]interface{}
func GetAll() Config {
	result := make(Config)
	for k, c := range configs {
		// if its an AutoConfigurator, use its c field
		if ac, ok := c.(AutoConfigurator); ok {
			result[k] = ac.c
		} else {
			result[k] = c
		}

	}
	return result
}

func GetModule(module string) Configurator {
	if c, ok := configs[module]; ok {
		return c
	}
	return nil
}


// Hooks registered here will be executed after the config package has finished
// loading the conf
func RegisterConfReadyHooks(hooks ...Hook) {
	ConfReadyHooks = append(ConfReadyHooks, hooks...)
}

// A call to this func will run all registered config hooks
func RunConfHooks(c *cli.Context) {
	log.Debug("running config hooks")
	for _, f := range ConfReadyHooks {
		err := f(c)
		if err != nil {
		  log.Fatalf("error running config hook: %v", err)
		}
	}
}

// A configurator can set options available under it's own module scope
// or under the global scope. A configurator implements the Configurator interface
func RegisterConfigurator(name string, c Configurator) {
	log.Debugf("Registering configurator %s", name)
	configs[name] = c
}

func init() {
	// Initialize the global config
	configs[GlobalConfigName] = make(Config)
}
