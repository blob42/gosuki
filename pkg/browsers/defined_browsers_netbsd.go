// Code generated DO NOT EDIT.

//go:build netbsd
package browsers

var DefinedBrowsers = []BrowserDef{
	{
		"chrome",
		1,
		"~/.config/google-chrome",
		"",
		"",
	},
	{
		"chromium",
		1,
		"~/.config/chromium",
		"",
		"",
	},
	{
		"librewolf",
		0,
		"~/.librewolf",
		"",
		"",
	},
	{
		"firefox",
		0,
		"~/.mozilla/firefox",
		"",
		"",
	},
	{
		"qutebrowser",
		2,
		"~/.config/qutebrowser",
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
