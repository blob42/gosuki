<p align="center">
<img src="internal/webui/static/favicon.svg" height=50>
<h3 align="center">GoSuki</h3>
<h4 align="center">Multi-browser, realtime, bookmark manager</h4>
<h5 align="center">No subscription or extension required</h5>

 <h4 align="center">
  <a href="#-in-action">Demo</a> |
  <a href="https://gosuki.net/docs/getting_started/quickstart">Documentation</a> |
  <a href="https://gosuki.net/">Website</a>
</h4>
 <h5 align="center">
  <a href="#installation">Install</a> |
  <a href="#-features">Feautres</a> |
  <a href="https://github.com/blob42/gosuki/blob/master/docs/faq.md">Faq</a>
</h5>


<br>
</p>


<h3 align="center">What's this ? Oh, just another bookmark manager. No big deal.</h3>

- [x] Ever feel like your bookmarks are a [chaotic mess](https://news.ycombinator.com/item?id=13734253) ?
- [x] Ever find yourself swapping browsers when they [stop meeting](https://github.com/uBlockOrigin/uBlock-issues/wiki/About-Google-Chrome's-%22This-extension-may-soon-no-longer-be-supported%22) your demands ?
- [x] Have past bookmark managers [let you down](https://support.mozilla.org/en-US/kb/future-of-pocket), vendor locking or abandoning you in your time of need ?
- [x] Would you rather avoid entrusting your bookmarks to cloud companies and [browser extensions](https://arstechnica.com/security/2025/07/browser-extensions-turn-nearly-1-million-browsers-into-website-scraping-bots/) ?
- [x] Perhaps you keep multiple browser profiles for work, research, streaming, and development ?
- [x] Do you use some [“obscure”](https://github.com/qutebrowser/qutebrowser) browser that doesn't support extensions ?

- [ ] If you're nodding your head to any of the above, then look no further:

**GoSuki** is a cloudless, **extension-free**, **multi-browser**, and **real time** bookmark tracker and organizer. It packs everything in a **single binary** and captures all your bookmarks in a **portable database** with [multi-device sync](https://gosuki.net/docs/features/multi-device-sync/) & [archiving][1] capability. It will eventually support [every internet browser under the sun](#how-does-it-work-).
<br>
<br>


<div align="center">
  <!-- Mozilla Firefox Browsers -->
  <a href="https://gosuki.net/docs/features/browser-support-matrix/" target="_blank"><img src="https://upload.wikimedia.org/wikipedia/commons/7/7a/Firefox_brand_logo%2C_2019.svg" width="30"></a>
  <a href="https://gosuki.net/docs/features/browser-support-matrix/" target="_blank"><img src="https://upload.wikimedia.org/wikipedia/commons/d/d0/LibreWolf_icon.svg" width="30"></a>
  <a href="https://gosuki.net/docs/features/browser-support-matrix/" target="_blank"><img src="https://upload.wikimedia.org/wikipedia/commons/e/e4/Waterfox_logo_2019.svg" width="30"></a>
  <a href="https://gosuki.net/docs/features/browser-support-matrix/" target="_blank"><img src="https://gosuki.net/images/browsers/palemoon_hu_926375eb54d69976.webp" width="30"></a>
  <a href="https://gosuki.net/docs/features/browser-support-matrix/" target="_blank"><img src="https://gosuki.net/images/browsers/icecat_hu_52ad98a42302fd41.webp" width="30"></a>
  <a href="https://gosuki.net/docs/features/browser-support-matrix/" target="_blank"><img alt="Zen browser" src="https://avatars.githubusercontent.com/u/165225155?s=200&v=4" width="30"></a>
  <a href="https://gosuki.net/docs/features/browser-support-matrix/" target="_blank"><img src="https://gosuki.net/images/browsers/basilisk_hu_9d66460a001b2264.webp" width="30"></a>
  <br>
  <!-- Chromium Browsers -->
  <a href="https://gosuki.net/docs/features/browser-support-matrix/" target="_blank"><img src="https://upload.wikimedia.org/wikipedia/commons/e/e1/Google_Chrome_icon_%28February_2022%29.svg" width="30"></a>
  <a href="https://gosuki.net/docs/features/browser-support-matrix/" target="_blank"><img src="https://upload.wikimedia.org/wikipedia/commons/2/28/Chromium_Logo.svg" width="30"></a>
  <a href="https://gosuki.net/docs/features/browser-support-matrix/" target="_blank"><img src="https://qutebrowser.org/icons/qutebrowser.svg" width="30"></a>
  <a href="https://gosuki.net/docs/features/browser-support-matrix/" target="_blank"><img src="https://upload.wikimedia.org/wikipedia/commons/1/1f/Brave-logo.svg" width="60"></a>
  <br>
  <a href="https://gosuki.net/docs/features/browser-support-matrix/" target="_blank"><img src="https://img.shields.io/badge/linux-gray?logo=linux"></a>
  <a href="https://gosuki.net/docs/features/browser-support-matrix/" target="_blank"><img src="https://img.shields.io/badge/macOS-gray"></a>
  <a href="https://gosuki.net/docs/features/browser-support-matrix/" target="_blank"><img src="https://img.shields.io/badge/Windows-0078D6?logo=windows"></a>
  <a href="https://gosuki.net/docs/features/browser-support-matrix/" target="_blank"><img src="https://img.shields.io/badge/FreeBSD-AB2B28?logo=freebsd"></a>
  <a href="https://gosuki.net/docs/features/browser-support-matrix/" target="_blank"><img src="https://img.shields.io/badge/NetBSD-FFC000?logo=netbsd"></a>
  <a href="https://gosuki.net/docs/features/browser-support-matrix/" target="_blank"><img src="https://img.shields.io/badge/OpenBSD-F5C400?logo=openbsd"></a>
  <a href="https://gosuki.net/docs/features/browser-support-matrix/" target="_blank"><img src="https://img.shields.io/badge/flatpak-blue?logo=flathub"></a>
  <a href="https://gosuki.net/docs/features/browser-support-matrix/" target="_blank"><img src="https://img.shields.io/badge/snap-orange?logo=snapcraft"></a>
</div>
<p align="center"><small><a href="https://gosuki.net/docs/contributing/browsers" target="_blank"> add your browser</a></small></p>


## ✨ Features

- 📦 **Standalone**: Gosuki is a single binary with no dependencies or external extensions necessary. It's designed to just work right out of the box
- ⌨️ **Ctrl+D**: Use the universal shortcut to add bookmarks and call [custom commands](https://gosuki.net/docs/features/marktab-actions)
- 🏷️ **Tag Everything**: Tag with **#hashtags** even if your browser does not support it. You can even add tags in the Title. Your folders become tags
- 🔎 **Real time**: Gosuki keeps track of your bookmarks, spotting any changes as they happen
- 📱 **Multi-Device-Sync**: [Synchronize](https://gosuki.net/docs/features/multi-device-sync/) your bookmarks across multiple devices.
- 🗃️ **Archiving** Archive your bookmarks with [ArchiveBox][1].
- 🖥️ **Web UI + CLI** Builtin, local Web UI. Also works without Javascript. dmenu/rofi compatible CLI.
- 🧪 **Hackable**: Modular and extensible. Custom scripts and actions per tags and folders.
- 🌎 **Browser Agnostic**: Detects which browsers you have installed and watch changes in all of them
- 👤 **Profile Support**: Also handles multiple profiles for each browser
- 💾 **Buku Compatibility**: Gosuki is compatible with the [Buku](https://github.com/jarun/buku) sqlite database, you can use any program [that was made for buku](https://github.com/jarun/buku?tab=readme-ov-file#related-projects)
- 📡 **External APIs** Consolidate your curated content from external APIs (github, reddit ...)


## 📸 In Action

<div align="center">
  <p>
    <h3><a href="https://github.com/user-attachments/assets/bb5c52f8-4413-4f91-88c7-445834728952">Realtime Bookmark Tracker</a></h3>
    <video controls muted src="https://github.com/user-attachments/assets/bb5c52f8-4413-4f91-88c7-445834728952"></video>
  </p>


  <p>
    <h3><a href="https://github.com/user-attachments/assets/2e69940a-2fc3-4108-9b4c-ef324b3d08cd">Marktab Scripts</a></h3>
    <video controls muted src="https://github.com/user-attachments/assets/2e69940a-2fc3-4108-9b4c-ef324b3d08cd"></video>
    <p>Note: you can also drop bookmarks in a folder matching an action to execute the script. Folders are tags</p>
  </p>

    
  <p>
    <h3><a href="https://github.com/user-attachments/assets/bf1e7c87-5775-4c54-a428-cfe84757c43e">CLI - Suki</a></h3>
    <video controls muted src="https://github.com/user-attachments/assets/bf1e7c87-5775-4c54-a428-cfe84757c43e"></video>
  </p>

</div>

<br>
<p align="center"><a href="https://youtu.be/lxrzR4cHgmI" target="_blank">Full Demo on YT</a></p>

## Installation


Checkout the [quick start guide](https://gosuki.net/docs/getting_started/quickstart).

Gosuki supports Linux, macOS, Windows, and BSD (FreeBSD, NetBSD, OpenBSD).

### Packages

<div>
<a href="https://aur.archlinux.org/packages/gosuki-git"><img src="https://img.shields.io/badge/Arch-AUR-blue?logo=archlinux"></a>
<br>
<a href="https://git.blob42.xyz/gosuki.net/-/packages/debian/gosuki"><img src="https://img.shields.io/badge/Debian-A81D33?logo=debian"></a>
<a href="https://git.blob42.xyz/gosuki.net/-/packages/debian/gosuki"><img src="https://img.shields.io/badge/Ubuntu-E95420?logo=ubuntu"></a>
<br>
<a href="https://github.com/blob42/homebrew-gosuki"><img src="https://img.shields.io/badge/Homebrew-gosuki-FBB040?logo=homebrew&logoColor=white"></a>
</div>


### From source

##### Dependencies:

- `sqlite3` development library

```shell
go install github.com/blob42/gosuki/cmd/gosuki@latest
```

- Build with systray icon feature

```shell
go install -tags systray github.com/blob42/gosuki/cmd/gosuki@latest
```

#### optional `suki` cli command

`suki` is a cli command to list/filter bookmarks with a customizable dmenu/rofi compatible output

```shell
go install github.com/blob42/gosuki/cmd/suki@latest
```

## Running GoSuki

GoSuki is designed to run as a background service for real-time bookmark monitoring. Below are the recommended ways to start and interact with the application.

### As a Service
Start GoSuki as a persistent service ([systemd example](contrib/gosuki.service)):
```bash
gosuki start
```
This initializes all configured browsers and begins real-time bookmark tracking.

By default the Web UI binds to `127.0.0.1:2025`. Use `--listen` to change the address:
```bash
gosuki --listen 0.0.0.0:8080 start
```

### Terminal UI (TUI)
Launch with an interactive terminal interface for real-time parsing overview:
```bash
gosuki --tui start
```
The TUI displays module status and bookmark processing metrics.

### Importing bookmarks

#### From Buku

```shell
gosuki import buku
```

This will imports all bookmarks from Buku into Gosuki. Gosuki DB is compatible with buku, meaning you can symlink gosuki DB or add it to Buku and it will just work. However, gosuki cannot read a buku database unless it is imported with the above command.

#### From pocket

Import bookmarks from a Pocket export in `csv` format:

```shell
gosuki import pocket export_file.csv
```

### Debugging
A leveled logging system is available with `--debug={trace,debug,info,warn,error,fatal,none}`

```bash
gosuki --debug=info 

## Set debug level per subsystem
gosuki --debug=info,chrome=trace
```

*note* use `--debug=list` to list the available levels and subsystems

## How does it work ?

Gosuki monitors the browser's internal bookmark files for changes. It uses the native OS event notification system to detect changes as they happen. This allows it to be fast and efficient, without the need for any plugins or extensions. This architecture makes it possible to load bookmarks from any kind of internet browser, API or program that emits something akin to a bookmark.

Browser definitions are stored in a single YAML file (`pkg/browsers/browsers.yaml`) and code is generated via `make gen`. Adding a new browser only requires adding its paths to this file.

The application maintains a **local** portable database of all tracked bookmarks, accessible via the built-in web UI or CLI.

Curious for more details on the internals ? Checkout the [Architecture](docs/technical/architecture.md) file.

## Rationale

I spent years working on and off on this project. My goal was to create a bookmark management solution resilient to vendor lock-in and the increasing trend of subscription services seeking rent for access to our curated internet content.

In the age of the "everything internet" web links and bookmarks represent a significant investment of human time spent curating and selecting relevant content. The past decade has seen a noticeable ~enshittification~ decline in the quality of internet results, with SEO-optimized blogs, marketing materials, and censored links making it increasingly difficult to find valuable websites and articles. Now with the advent of AI-generated “slop” and low-quality content, we face an  endless stream of potentially  fake and unreliable information.

GoSuki is my modest attempt to make the definitive solution for managing internet bookmarks that is: browser and platform agnostic, immune to vendor lock-in and fully self contained. The extension-free architecture of gosuki means that eventually every internet browser under the sun will be supported. 

If you like this project consider starring the repo and [supporting the effort](#support).


## Roadmap

- [x] **Multi-device Sync** - [Synchronization between multiple devices](https://gosuki.net/docs/features/multi-device-sync/)
- [x] **Archiving** - Archive bookmarks with [ArchiveBox][1]
- [ ] **Linkrot** - Automatically identify broken links and replace with web.archive.org alternatives
- [ ] **Metadata Refresh** - Automatically clean and update tags/metadata for existing bookmarks
- [ ] **Management UI** - Self-hosted server node with admin interface.
- [ ] **Browser Sync** - Push changes back to browsers for consistent bookmark management
- [ ] **Packaging** - More distros and FreeBSD ports
- [ ] **More Platforms** - Android 

## Support

GoSuki is a one-man project. If you find value in this tool, consider supporting its development through:

- Reporting issues and suggesting features
- Testing and [adding new browsers](https://gosuki.net/docs/contributing/browsers)
- Creating modules for third-party APIs
- Contributing code or documentation
- Sharing the project with others who might benefit
- Financial contributions via [GitHub Sponsors](https://github.com/sponsors/blob42) or [Patreon](https://www.patreon.com/c/GoSuki)


## Contributing

We welcome contributions from the community! To get started:

1. Fork the repository
2. Create a new branch for your changes
3. Submit a pull request with clear documentation

For bug reports, please provide detailed steps to reproduce the issue.


## Related Projects 

Read the ["how does it compare to"](docs/how-does-it-compare-to.md) guide.

- [Buku](https://github.com/jarun/buku): Gosuki is compatible with Buku
- [Shiori](https://github.com/go-shiori/shiori): Simple bookmark manager built with Go
- [bmm](https://github.com/dhth/bmm): Get to your bookmarks in a flash
- [wallabag](https://github.com/wallabag/wallabag): Self-hosted application for saving web pages
- [floccus](https://floccus.org/): Self-hosted extension based bookmark synchronization

## Links & Discussions

- [Ask HN: Do you still use browser bookmarks?](https://news.ycombinator.com/item?id=14064096)
- [Ask HN: How to handle bookmarks so you can find them again?](https://news.ycombinator.com/item?id=13734253)
- [Reddit: Does anyone actually use mobile bookmarks](https://www.reddit.com/r/firefox/comments/dez7hh/does_anyone_actually_use_mobile_bookmarks/)
- [You are the dead internet](https://www.youtube.com/watch?v=aoTQPoz9_As)
- [How to choose a browser for everyday use ?](https://digdeeper.neocities.org/articles/browsers.xhtml)
- [Lifting the veil - how to test browsers for spyware](https://digdeeper.neocities.org/articles/liftingtheveil.xhtml)

---

> In memory of my father, mentor, role model, best friend, and the kindest person I have ever known.

[1]:https://gosuki.net/docs/features/archiving/archive-box
