# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [unreleased]

### Added

- ArchiveBox integration using @archivebox tag or folder 
- internal: call bookmark hooks from anywhere using bit flags
- new export formats: pocket-html, json(pinboard/wallabag), generic XML-RSS
- `suki search` command
- `sudki fuzzy|search [term...] :tag1,tag2` to filter by tags with logical AND
- `sudki fuzzy|search [term ...] :OR tag1,tag2` to filter by tags with logical OR
- updated queries and api to filter by query and many tags

### Fixed

- html-autoimport: parse hash tags from title 

## [1.2.1] 2025-08-09

### Added 

- Waterfox browser support thanks to @weiss-d 

### Fixed

- broken default db path on startup

## [1.2.0] 2025-08-07

### Added

- tui: improved tui ux: toggle logs, expand/collapse details, total db bookmarks
- cli: export to netscape html format
- cli: `--db` flag parameter to set a custom path for gosuki database
- browsers: ability to set a custom profile path for firefox/chrome based browsers
- message/goroutine based inter-module communication 

### Changed

- config: global config options moved to root of toml file
- cli: improved `gosuki profile list` and `gosuki modules list` commands
- upgraded to schema v3: introduced the `version` and `node_id` on the `gskbookmarsk` table that is used by multi-device synchronization
- refactored logging to allow setting levels per subsystem
- added Trace level for very noisy log

### Fixed

- cli: many fixes to cli flag behavior
- enable generated imports modules in /mods package (enables mods for installs from source)
- Log browser profile path errors as warnings
- Reduce log verbosity on default level

## [1.1.0] 2025-07-29

### Added

- github module: periodic import of starred repos
- import bookmarks from Pocket csv export with `buku import pocket`
- cli: `--silent` and `-S` to fully disable any log
- Added support for brave browser (linux, snap, flatpak)
- Flatpak support for: google-chrome, chromium, firefox, librewolf
- Database schema upgrade to v2 with `xhsum` column for efficient synchronization and conflict resolution
- Two-level database cache (L1/L2) for improved performance and consistent data state
- CLI command `buku import` for importing a buku DB to gosuki
- Schema versioning tracking in `schema_version` table
- `bookmarks` view with INSTEAD OF triggers for Buku compatibility
- Example bookmark launcher with rofi `contrib/rofi.sh`

### Changed

- `gosuki buku import` becomes `gosuki import buku`
- Refactored cli to use urfave cli v3
- BREAKING: Database schema modified to allow future upgrades
- Schema migration: `gskbookmarks` table replaces `bookmarks` (legacy `bookmarks` remains as a view)
- Hide helper script from public doc

### Fixed

- UpsertBookmark: does not unset the title if the new value is empty
- Description field not being updated in some cases
- CLI: use custom path to config file
- CLI: fix watch-all flag
- TUI: disable tui code in daemon mode


##### Changes to DB Schema 

Previously there was only a `bookmarks` table which contained a
few extra column compared to Buku. This table is renamed to
`gskbookmarks` which will be the native gosuki table schema. 

Instead we provide a `bookmarks` view into `gskbookmarks` with
INSERT and UPDATE trigger (INSTEAD OF) that allow Buku programs to
use Gosuki DB as a buku database while still benifiting from
gosuki specific features that will eventually be added to the
schema.

Added also schema versioning that will be tracked in the table
creatively named `schema_version`.


## [1.0.0] - 2025-12-07

### Added

- example bookmark launcher with rofi `contrib/rofi.sh`

### Fixed

- cli: use custom path to config file
- cli: fix watch-all flag
- tui: disable tui code in daemon mode


## [1.0.0-rc1] - 2025-07-07

Initial public release

[unreleased]: https://github.com/blob42/gosuki/compare/v1.2.1...HEAD
[1.2.1]: https://github.com/blob42/gosuki/releases/tag/v1.2.1
[1.2.0]: https://github.com/blob42/gosuki/releases/tag/v1.2.0
[1.1.0]: https://github.com/blob42/gosuki/releases/tag/v1.1.0
[1.0.0]: https://github.com/blob42/gosuki/releases/tag/v1.0.0
[1.0.0-rc1]: https://github.com/blob42/gosuki/releases/tag/v1.0.0-rc1
