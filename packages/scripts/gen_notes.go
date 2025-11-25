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

// script that generates gosuki release notes from a template
package main

import (
	"flag"
	"html/template"
	"io"
	"os"
)

var (
	changelog    = flag.String("changelog", "{{.changelog}}", "path to the changelog file")
	summary      = flag.String("summary", "{{.summary}}", "release summary")
	debVersion   = flag.String("deb-version", "{{.debVersion}}", "debian package version")
	contributors = flag.String("contributors", "{{.contributors}}", "contributors list")
)

func main() {
	flag.Parse()

	// releases template
	tplStr, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	tmpl, err := template.New("release").Parse(string(tplStr))
	if err != nil {
		panic(err)
	}

	data := map[string]any{
		"summary":      *summary,
		"debVersion":   *debVersion,
		"contributors": *contributors,
	}

	data["changelog"] = "{{.changelog}}"
	if changelog != nil {
		// Test if changelog is a valid path
		if _, err := os.Stat(*changelog); err == nil {
			// Read all content from the file
			content, err := os.ReadFile(*changelog)
			if err == nil && len(content) > 0 {
				data["changelog"] = string(content)
			}
		}
	}

	err = tmpl.Execute(os.Stdout, data)
	if err != nil {
		panic(err)
	}
}
