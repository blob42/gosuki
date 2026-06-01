// Code generated DO NOT EDIT.

//go:build darwin
package browsers

var DefinedBrowsers = []BrowserDef{
	{
		"brave",
		1,
		"~/Library/Application Support/BraveSoftware/Brave-Browser",
		"",
		"",
	},
	{
		"chrome",
		1,
		"~/Library/Application Support/Google/Chrome",
		"",
		"",
	},
	{
		"chromium",
		1,
		"~/Library/Application Support/chromium",
		"",
		"",
	},
	{
		"basilisk",
		0,
		"~/Library/Application Support/Basilisk",
		"",
		"",
	},
	{
		"firefox",
		0,
		"~/Library/Application Support/Firefox",
		"",
		"",
	},
	{
		"librewolf",
		0,
		"~/Library/Application Support/Librewolf",
		"",
		"",
	},
	{
		"palemoon",
		0,
		"~/Library/Application Support/PaleMoon",
		"",
		"",
	},
	{
		"zen",
		0,
		"~/Library/Application Support/zen",
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
