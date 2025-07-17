# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Added support for brave browser (linux, snap, flatpak)
- Flatpak support for: google-chrome, chromium, firefox, librewolf
- Database schema upgrade to v2 with `xhsum` column for efficient synchronization and conflict resolution
- Two-level database cache (L1/L2) for improved performance and consistent data state
- CLI command `buku import` for importing a buku DB to gosuki
- Schema versioning tracking in `schema_version` table
- `bookmarks` view with INSTEAD OF triggers for Buku compatibility
- Example bookmark launcher with rofi `contrib/rofi.sh`

### Changed

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

[unreleased]: https://github.com/blob42/gosuki/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/blob42/gosuki/releases/tag/v1.0.0
[1.0.0-rc1]: https://github.com/blob42/gosuki/releases/tag/v1.0.0-rc1
