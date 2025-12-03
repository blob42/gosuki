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

//go:build darwin

package browsers

var DefinedBrowsers = []BrowserDef{
	Firefox,
	Librewolf,
	Zen,
	Chrome,
	Chromium,
	QuteBrowser,
}

// Other Browsers
var (
	QuteBrowser = BrowserDef{
		"qutebrowser",
		Qutebrowser,
		"~/Library/Application Support/qutebrowser",
		"", "",
	}
)

// Chrome Browsers
var (
	Chrome = ChromeBrowser(
		"chrome",
		"~/Library/Application Support/Google/Chrome",
		"", "",
	)
	Chromium = ChromeBrowser(
		"chromium",
		"~/Library/Application Support/chromium",
		"", "",
	)
)

// Mozilla Browsers
var (
	Firefox = MozBrowser(
		"firefox",
		"~/Library/Application Support/Firefox",
		"", "",
	)

	Librewolf = MozBrowser(
		"librewolf",
		"~/Library/Application Support/Librewolf",
		"", "",
	)

	Zen = MozBrowser(
		"zen",
		"~/Library/Application Support/zen",
		"", "",
	)
)

func AddBrowserDef(b BrowserDef) {
	DefinedBrowsers = append(DefinedBrowsers, b)
}
