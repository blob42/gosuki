// Code generated DO NOT EDIT.

//go:build linux
package browsers

var DefinedBrowsers = []BrowserDef{
	{
		"chrome",
		1,
		"~/.config/google-chrome",
		"",
		"~/.var/app/com.google.Chrome/config/google-chrome",
	},
	{
		"chromium",
		1,
		"~/.config/chromium",
		"~/snap/chromium/common/chromium/",
		"~/.var/app/org.chromium.Chromium/config/chromium",
	},
	{
		"brave",
		1,
		"~/.config/BraveSoftware/Brave-Browser",
		"~/snap/brave/current/.config/BraveSoftware/Brave-Browser",
		"~/.var/app/com.brave.Browser/config/BraveSoftware/Brave-Browser",
	},
	{
		"librewolf",
		0,
		"~/.librewolf",
		"",
		"~/.var/app/io.gitlab.librewolf-community/.librewolf",
	},
	{
		"waterfox",
		0,
		"~/.waterfox",
		"",
		"~/.var/app/net.waterfox.waterfox/.waterfox",
	},
	{
		"icecat",
		0,
		"~/.mozilla/icecat",
		"",
		"",
	},
	{
		"zen",
		0,
		"~/.zen",
		"",
		"~/.var/app/app.zen_browser.zen/.zen",
	},
	{
		"palemoon",
		0,
		"~/.palemoon",
		"",
		"",
	},
	{
		"basilisk",
		0,
		"~/.basilisk",
		"",
		"",
	},
	{
		"firefox",
		0,
		"~/.mozilla/firefox",
		"~/snap/firefox/common/.mozilla/firefox",
		"~/.var/app/org.mozilla.firefox/.mozilla/firefox",
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
