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

//go:build linux

package browsers

var DefinedBrowsers = []BrowserDef{
	Firefox,
	Librewolf,
	Chrome,
	Chromium,
	Brave,
	QuteBrowser,
}

// Other Browsers
var (
	QuteBrowser = BrowserDef{
		"qutebrowser",
		Qutebrowser,
		"~/.config/qutebrowser",
		"/nonexistent",
		"/nonexistent",
	}
)

// Chrome Browsers
var (
	Chrome = ChromeBrowser(
		"chrome",
		"~/.config/google-chrome",
		"/nonexistent",
		"~/.var/app/com.google.Chrome/config/google-chrome",
	)
	Chromium = ChromeBrowser(
		"chromium",
		"~/.config/chromium",
		"~/snap/chromium/common/chromium/",
		"~/.var/app/org.chromium.Chromium/config/chromium",
	)
	Brave = ChromeBrowser(
		"brave",
		"~/.config/BraveSoftware/Brave-Browser",
		"~/snap/brave/current/.config/BraveSoftware/Brave-Browser",
		"~/.var/app/com.brave.Browser/config/BraveSoftware/Brave-Browser",
	)
)

// Mozilla Browsers
var (
	Firefox = MozBrowser(
		"firefox",
		"~/.mozilla/firefox",
		"~/snap/firefox/common/.mozilla/firefox",
		"~/.var/app/org.mozilla.firefox/.mozilla/firefox",
	)

	Librewolf = MozBrowser(
		"librewolf",
		"~/.librewolf",
		"/nonexistent",
		"~/.var/app/io.gitlab.librewolf-community/.librewolf",
	)

	Waterfox = MozBrowser(
		"waterfox",
		"~/.waterfox",
		"/nonexistent",
		"~/.var/app/net.waterfox.waterfox",
	)
)

func AddBrowserDef(b BrowserDef) {
	DefinedBrowsers = append(DefinedBrowsers, b)
}
