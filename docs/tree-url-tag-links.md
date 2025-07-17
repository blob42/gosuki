# linking urls to tags and folders at the same time 

options:

1. store every bookmark under a tag node as well as a corresponding folder nodes
   
   - a lot of code changes required, Node.Parent would point to multiple parents
   - will break tree algorithms or not ? it should not break
   
   1.2 store url nodes under tag nodes but keep their parent as folers only

2. Store a pointer to tags for each node
   - if node has no tags (folders) the tags node list is empty
   - require change of Node.Tags from []string to []*Node

   ```

     | PrintTree
     | .
     | └── 0 <root>
     |     ├── 2 <Bookmarks Menu>
     |     │   ├── 2 <Mozilla Firefox>
     |     │   └── 2 <GosukiMenu>
     |     ├── 2 <Bookmarks Toolbar>
     |     │   ├── 2 <cooking>
     |     │   │   └── 2 <indian>
     |     │   └── 2 <Travel>
     |     ├── 2 <Other Bookmarks>
     |     ├── 2 <Mobile Bookmarks>
     |     └── 0 <TAGS>
     |         ├── 3 <golang>
     |         │   └── 1 <The Go Programming Language>
     |         ├── 3 <programming>
     |         │   ├── 1 <The Go Programming Language>
     |         │   └── 1 <Rust Programming Language>
     |         ├── 3 <>
     |         │   └── 1 <Indian Cooking at Home: A Beginner's Guide | Taste of 
Home>
     |         ├── 3 <based>
     |         │   └── 1 <Based Cooking>
     |         ├── 3 <rust>
     |         │   └── 1 <Rust Programming Language>
     |         ├── 3 <systems>
     |         │   └── 1 <Rust Programming Language>
     |         ├── 3 <budapest>
     |         │   └── 1 <Budapest - Official tourist information >
     |         ├── 3 <gosuki>
     |         │   └── 1 <universal bookmark tracker>
     |         └── 3 <libre>
     |             └── 1 <Front Page — Free Software Foundation — working togeth
er for free software>
```


## References to Node.Tags:

firefox/firefox.go|265 col 43|   ./firefox/firefox.go                                           Tags Reference 6 of: 11
firefox/firefox.go|265 col 43| 265:       urlNode.Tags = append(urlNode.Tags, tagNode.Name)      碑 () scanBookmarks()
firefox/firefox.go|536 col 15| 536:   urlNode.Tags = utils.Extends(urlNode.Tags, tagNode.Name)             碑 () Run()
firefox/firefox.go|787 col 33| 787: urlNode.Tags = append(urlNode.Tags, tagNode.Name)            碑 loadBookmarks()
parsing/parse.go|55 col 8|   ./parsing/parse.go                                                       Reference 3 of: 11
parsing/parse.go|55 col 8| 55: node.Tags = append(node.Tags, m[1])                                    碑 ParseTags()
parsing/parse.go|59 col 14| 59: if len(node.Tags) > 0                                                   ParseTags()
parsing/parse.go|60 col 58| 60: log.Debugf("[in title] found following tags: %s", node.Tags)            ParseTags()
tree/tree.go|31 col 2|   ./tree/tree.go                                                          Reference 1
tree/tree.go|31 col 2| 31: Tags       []string                                                              什 
