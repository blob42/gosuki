# Buffers and Cache

### Cache
- The **Cache** in code is `cacheDB` and uses **sqlite**
- It represents the bookmarks over all browsers.
- It is periodically synced to the local disk gosuki.
- It is preloaded to memory when the program is started.

### Buffer
- is an **sqlite** memory db called `bufferDB`
- Represents *one browser* state across all jobs
- Is synced with `cacheDB`

### URLIndex
This is a [RedBlack Tree Hashmap](https://github.com/sp4ke/hashmap).

- It holds in memory the last state of the bookmark tree.
- Used as fast query db
- Each URL holds a pointer to a node in [nodeTree](#nodeTree)


### nodeTree

- Is a tree datastructure that can be stored in a browser representing the bookmarks in a node structure. 
- It allows for easy parsing of Folders and URLs and keeps the parent/child relationship.


## Architecture

- Insertion happens when 
  1. New bookmarks are detected
  2. Initial loading of bookmarks when the program starts

- The index needs to have very fast lookups, since every bookmark is checked to see if the tags are modified.
 
-The test process needs to be as fast as possible, the best being to create a fast hash of the data to test in the index first time when the browser bookmarks are loaded. The hash is tested again when a reload of the bookmarks is done.

## Data structures

### Hash Map 
[hash map](https://github.com/DusanKasan/hashmap)
- The hash map is a map of [hash_url] --> hash_content.
- the URL becomes the main index for lookups
- The `hash_content` is the hash of content to test against if it changed. 

### Hash function
[xxhash](https://github.com/OneOfOne/xxhash)
- This is the hash function used to generate the url_hash as well as the content hash.


# Parsing Bookmarks and Tags

- Run program
- Preload bookmark to `cacheDB`
- Bookmarks changed event
- Create a new `nodeTree building job` and for each bookmark do:
  - If BK not in URLIndex continue, add to Index as pointer to BK node
  - If BK in URLIndex and name changed, mark BK as `has_changed`
  - Run parsing hooks on BK
- Rebuild the Index to mirror the nodeTree
- Sync the `nodeTree` to `bufferDB`
- Sync `bufferDB` to `cacheDB`
- Flush cache to disk
