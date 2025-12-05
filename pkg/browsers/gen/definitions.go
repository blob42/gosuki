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

package main

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/blob42/gosuki/pkg/browsers"
)

const base_tpl = `// Code generated DO NOT EDIT.

//go:build {{.platform}}
package browsers

var DefinedBrowsers = []BrowserDef{
	{{- range .defs }}
	{
		"{{.Flavour}}",
		{{.Family}},
		"{{printf "%s" .BaseDir}}",
		"{{printf "%s" .SnapDir}}",
		"{{printf "%s" .FlatpakDir}}",
	},{{ end }}
}

func Defined(family BrowserFamily) map[string]BrowserDef {
	result := map[string]BrowserDef{}
	for _, bd := range DefinedBrowsers {
		if bd.Family == family {
			result[bd.Flavour] = bd
		}
	}

	return result
}

func AddBrowserDef(b BrowserDef) {
	DefinedBrowsers = append(DefinedBrowsers, b)
}
`

type platform string
type browserConfigs map[platform][]browsers.BrowserDef

// generates browsers definitions from yaml
// defFile is the path to a yaml file
// open and parse the file
func generateBrowserConfs(defFile string) (browserConfigs, error) {
	data, err := os.ReadFile(defFile)
	if err != nil {
		return nil, err
	}

	var cfg browsers.BrowserConfig
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// pretty.Println(cfg)

	bCfgs := make(map[platform][]browsers.BrowserDef)

	// Chrome browsers
	for flavour, platforms := range cfg.Chrome {
		for p, pCfg := range platforms {
			bCfgs[platform(p)] = append(bCfgs[platform(p)], browsers.ChromeBrowser(
				string(flavour),
				pCfg.BaseDir,
				pCfg.Snap,
				pCfg.Flatpak,
			))
		}
	}

	for flavour, platforms := range cfg.Mozilla {
		for p, pCfg := range platforms {
			if bCfgs[platform(p)] == nil {
				bCfgs[platform(p)] = []browsers.BrowserDef{}
			}
			bCfgs[platform(p)] = append(bCfgs[platform(p)], browsers.MozBrowser(
				string(flavour),
				pCfg.BaseDir,
				pCfg.Snap,
				pCfg.Flatpak,
			))
		}
	}

	// Custom browser defs
	for family, definitions := range cfg.Other {
		for flavour, platforms := range definitions {
			for p, pCfg := range platforms {
				bDef := browsers.BrowserDef{
					Flavour:    string(flavour),
					Family:     family,
					BaseDir:    pCfg.BaseDir,
					SnapDir:    pCfg.Snap,
					FlatpakDir: pCfg.Flatpak,
				}
				if bCfgs[platform(p)] == nil {
					bCfgs[platform(p)] = []browsers.BrowserDef{}
				}
				bCfgs[platform(p)] = append(bCfgs[platform(p)], bDef)
			}
		}
	}

	// pretty.Println(bCfgs)
	return bCfgs, nil
}

func generateBrowserDefs(confs browserConfigs, relPath string) error {
	var err error
	// pretty.Println(confs)

	tmpl := template.Must(template.New("linux_defs").Parse(base_tpl))

	for platform, pConfs := range confs {
		var buf bytes.Buffer
		tmpl.Execute(&buf, map[string]any{
			"platform": platform,
			"defs":     pConfs,
		})

		// fmt.Fprintf(os.Stdout, buf.String())
		// return nil
		defFile := fmt.Sprintf("defined_browsers_%s.go", platform)
		fmt.Printf("%s/%s\n", relPath, defFile)
		if err = os.WriteFile(defFile, buf.Bytes(), 0644); err != nil {
			return err
		}

	}
	return nil

}

// This is called from a go:generate directive
// It takes a single argument pointing to a file
// It creates the file path as PWD/file
func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "expected exactly one argument\n")
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get current directory: %v\n", err)
		os.Exit(1)
	}

	// Find the project root by traversing up until we find a go.mod file
	projectRoot := cwd
	for {
		if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
			break // found root
		}
		parentDir := filepath.Dir(projectRoot)
		if parentDir == projectRoot || parentDir == "/" {
			break // reached the filesystem root
		}
		projectRoot = parentDir
	}

	relPath, err := filepath.Rel(projectRoot, cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get relative path: %v", err)
	}
	filePath := cwd + "/" + os.Args[1]

	// fmt.Println(filePath)

	fmt.Printf("Generating browser definitions...\n")

	confs, err := generateBrowserConfs(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate browser definitions: %v\n", err)
	}

	if err = generateBrowserDefs(confs, relPath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate browser definitions: %v\n", err)
	}
	println()
}

// brs := []string{"chrome", "firefox", "qute"}
// tmpl := template.Must(template.New("browser_defs").Parse(base_tpl))
// tmpl.Execute(os.Stdout, brs)
//
// browsers.GenDefinitions()
