// Code generated DO NOT EDIT.

//go:build windows
package browsers

var DefinedBrowsers = []BrowserDef{
	{
		"brave",
		1,
		"%LOCALAPPDATA%\\BraveSoftware\\Brave-Browser\\User Data",
		"",
		"",
	},
	{
		"chrome",
		1,
		"%LOCALAPPDATA%\\Google\\Chrome\\User Data",
		"",
		"",
	},
	{
		"chromium",
		1,
		"%LOCALAPPDATA%\\Chromium\\User Data",
		"",
		"",
	},
	{
		"edge",
		1,
		"%LOCALAPPDATA%\\Microsoft\\Edge\\User Data",
		"",
		"",
	},
	{
		"firefox",
		0,
		"%APPDATA%\\Mozilla\\Firefox",
		"",
		"",
	},
	{
		"qutebrowser",
		2,
		"%APPDATA%\\qutebrowser",
		"",
		"",
	},
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
