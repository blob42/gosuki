// Copyright (c) 2025 Chakib Ben Ziane <contact@blob42.xyz>  and [`gosuki` contributors](https://github.com/blob42/gosuki/graphs/contributors).
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
package main

import (
	"context"
	"fmt"
	"go/build"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/tools/go/packages"
)

const targetPrefix = "./mods/"

func getGoMod() (string, error) {
	cmd := exec.Command("go", "list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func main() {
	module, err := getGoMod()
	if err != nil {
		log.Fatalf("error: %s", err)
	}
	// fmt.Println("module: ", module)

	ctx := context.Background()
	cfg := &packages.Config{
		Mode:    0,
		Context: ctx,
	}
	// pkg, err := packages.Load(cfg, module+"/mods/importer")
	// if err != nil {
	// 	log.Fatalf("error: %w", err)
	// }
	// pretty.Print(pkg)

	disabledRe := regexp.MustCompile(`\.disabled$`)

	header := `package mods
import (
`
	fmt.Print(header)
	err = filepath.WalkDir(targetPrefix, func(path string, d fs.DirEntry, err error) error {
		// fmt.Printf("%#v\n", path)
		// fmt.Printf("%#v\n", targetPrefix)

		if path == targetPrefix {
			return nil
		}

		// pretty.Print(d)
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}

		// skip disbaled modules
		if disabledRe.Match([]byte(d.Name())) {
			return filepath.SkipDir
		}

		rel, _ := filepath.Rel(path, targetPrefix)
		// fmt.Println("rel ", rel)
		if rel != ".." {
			return filepath.SkipDir
		}

		pkgs, err := packages.Load(cfg, filepath.Join(module, path))
		if err != nil {
			// Skip directories that are not valid Go packages
			if _, ok := err.(*build.NoGoError); ok {
				return nil
			}
			return fmt.Errorf("failed to import dir %s: %v", path, err)
		}

		for _, pkg := range pkgs {
			// pretty.Print(pkg)
			fmt.Printf("\t_ \"%s\"\n", pkg.ID)
		}

		return nil
	})
	fmt.Println(")")

	if err != nil {
		fmt.Fprintf(os.Stderr, "error walking the path %q: %v\n", targetPrefix, err)
		os.Exit(1)
	}
}
