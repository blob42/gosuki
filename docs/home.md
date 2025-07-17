## References and links:

### Buku bookmark manager

Buku is a python bookmark manager, it implements most of the algorithms and functions needed for our project. The`buku.py` file is included in the repository as a reference for functions to implement.

### Other links

- [Algorithms and data structures used](Algorithms-and-Data-Structres)

- [Fast json parsing library in golang (not depending on encoding/json)](https://github.com/buger/jsonparser)
- [Parsing and loading firefox/chrome bookmarks in buku](https://github.com/jarun/Buku/issues/175)
- [Sqlite indexes](https://www.tutorialspoint.com/sqlite/sqlite_indexes.htm) and [sqlite planner](https://www.sqlite.org/queryplanner.html)
- [Sqlite quick tutorial](http://tech.marksblogg.com/sqlite3-tutorial-and-guide.html)

#### Misc
- https://github.com/unode/firefox_decrypt

### Project Code Architecture

### Database types
#### Cache
Name: memcache

Used as memory buffer between Gosuki and the local gosuki database `gosuki.db`

#### Buffer
Name: buffer_<browser>

Used as memory buffer between gosuki and <browser> bookmarks
