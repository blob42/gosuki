# Development

## Dependencies

### 1. Make dependencies:

Use the Makefile to install dependencies with `make deps`.
*caddy v2* is needed as well

#### dependencies:
(arch linux)
- inotify-tools (watch file changes)
- libnotify (linux desktop notifications)


# Quick Intro

Gosuki is a blazing fast real time bookmarks sniffer and synchronizer.

It detects installed browsers in the system and automatically watches for
bookmark changes without relying on any external plugin. This is achieved by
manually reading the corresponding bookmark files. This solution allows for a
fully passive way to copy bookmarks without installing any plugin. 

# Design And Architecture

## Sync Strategy

Since gosuki might run against multiple running browsers, handling various
scenarios for when bookmarks are deleted would be very complex. The chosen
solution is to do *Read Only*  operations on the browser bookmark files. 

Any bookmark deleted on Gosuki database will not be synced back to the browser. 
Same for the bookmarks deleted on browser which is not carried to gosuki. This
means, gosuki is a read only backup for all bookmarks ever added on any
registered browser.

This should not be an issue as Gosuki provides its own Web Interface UI that
gives access to all gathered bookmarks from all browser. Modification/deletion
could be carried directly on that database.

*Note* that changes  on bookmarks  ARE detected and synced from the browser to
gosuki.

If a real need for syncing back bookmarks from Gosuki back to the browser is
needed, browser extensions could esailly be developped using gosuki API.

## The Gosuki database is compatible with BUKU

The sqlite3 database format used in gosuki was designed to be fully copatible
with [buku](https://github.com/jarun/Buku). This means:

- All bookmarks saved by gosuki can be directly accessed and manipulated using
  buku.

- All bookmarks saved by buku can be loaded in gosuki.

## Readings

### Tree Diff (Graph Isomorphism)

- [SO](https://stackoverflow.com/questions/5894879/detect-differences-between-tree-structures)
- [React Diffing](https://reactjs.org/docs/reconciliation.html)
- [Change Distilling](http://www.merlin.uzh.ch/contributionDocument/download/2162)
- [ A congruence theorem for trees ](https://msp.org/pjm/1957/7-1/p14.xhtml)
- [The Design and Analysis of Computer Algorithms](https://www.amazon.com/Design-Analysis-Computer-Algorithms/dp/0201000296)

idea: Rebuild the target tree on change ?


# Related - Libraries

- https://godoc.org/github.com/shirou/gopsutil (Process utils)
- [sqlite queries for common programs](https://github.com/kacos2000/Queries)

## Other Crawlers
- github shiori (golang)
- https://github.com/spyglass-search/netrunner (rust)
- https://github.com/a5huynh/spyglass (rust)
