// Copyright (c) 2023 Chakib Ben Ziane <contact@blob42.xyz>  and [`gosuki` contributors](https://github.com/blob42/gosuki/graphs/contributors).
// All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This file is part of GoSuki.
//
// GoSuki is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// GoSuki is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with gosuki.  If not, see <http://www.gnu.org/licenses/>.
package config


import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path"
	"slices"

	"github.com/BurntSushi/toml"
	"github.com/mitchellh/mapstructure"

	"github.com/blob42/gosuki/internal/utils"
)

const (
	ConfigFileName = "config.toml"
)

func getDefaultConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not get config dir: %s", err)
	}
	if configDir == "" {
		return "", errors.New("could not get config dir")
	}

	configDir = path.Join(configDir, utils.GosukiDirName)
	return configDir, nil
}

func getDefaultConfigPath() (string, error) {
	if configDir, err := getDefaultConfigDir(); err != nil {
		return "", err
	} else {
		return path.Join(configDir, ConfigFileName), nil
	}
}

func DefaultConfPath() string {
	configFile, err := getDefaultConfigPath()
	if err != nil {
		log.Fatal(err)
	}

	return configFile
}

func ConfigExists(path string) (bool, error) {
	return utils.CheckFileExists(path)
}

// Create a default toml config file
func createDefaultConfFile() error {
	var configDir string
	var err error

	if configDir, err = getDefaultConfigDir(); err != nil {
		return err
	}

	if err = os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("could not create config dir: %w", err)
	}

	configFilePath := path.Join(configDir, ConfigFileName)

	return createConfFile(configFilePath)
}

func MapToOutputConfig(in Config) (Config, error) {
	outMap := Config{}

	for k, v := range in {
		if k == GlobalConfigName {
			var globalConfig map[string]any
			if err := mapstructure.Decode(v, &globalConfig); err != nil {
				return nil, fmt.Errorf("failed to decode global config: %w", err)
			}
			maps.Copy(outMap, globalConfig)
		} else {
			outMap[k] = v
		}
	}
	return outMap, nil
}

func createConfFile(path string) error {
	var outConf Config
	configFile, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file %s: %w", path, err)
	}
	defer configFile.Close()

	if outConf, err = MapToOutputConfig(GetAll()); err != nil {
		return err
	}

	tomlEncoder := toml.NewEncoder(configFile)
	tomlEncoder.Indent = ""
	if err := tomlEncoder.Encode(&outConf); err != nil {
		return fmt.Errorf("failed to encode config to TOML: %w", err)
	}

	fmt.Printf("config written to %s\n", path)
	return nil
}

func LoadFromTomlFile(path string) error {
	buffer := make(Config)
	_, err := toml.DecodeFile(path, &buffer)
	if err != nil {
		return fmt.Errorf("loading config file %w", err)
	}

	// first map into global config
	if err := configs[GlobalConfigName].MapFrom(buffer); err != nil {
		return err
	}

	for k, val := range buffer {

		// only consider module configs
		if slices.Contains(slices.Collect(maps.Keys(configs)), k) {
			// send the conf to its own module
			if _, ok := configs[k]; !ok {
				configs[k] = make(Config)
			}
			err = configs[k].MapFrom(val)
		}

		// err = configs[k].MapFrom(val)
		if err != nil {
			return fmt.Errorf("parsing config <%s>: %w", k, err)
		}

	}
	return err
}
