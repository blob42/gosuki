// Script to generate a json file containg 1000 random bookmark.
// The json file format is as following:
//
//	{
//	  "checksum": "f66ad55299c757c5e400fbdc7905c5a1",
//	  "roots": {
//	     "bookmark_bar": {
//	        "children": [ {
//	           "date_added": "13152359724572936",
//	           "id": "9",
//	           "meta_info": {
//	              "last_visited_desktop": "13153403771745238"
//	           },
//	           "name": "Hacker News",
//	           "type": "url",
//	           "url": "https://news.ycombinator.com/"
//	        }, {
//	           "children": [ {
//	              "children": [ {
//	                 "date_added": "13152713849930471",
//	                 "id": "14",
//	                 "meta_info": {
//	                    "last_visited_desktop": "13152713849930745"
//	                 },
//	                 "name": "Chrome Web Store - Extensions",
//	                 "type": "url",
//	                 "url": "https://chrome.google.com/webstore/category/extensions?hl=en"
//	              } ],
//	              "date_added": "13152713751052410",
//	              "date_modified": "13152713865793130",
//	              "id": "13",
//	              "name": "Test",
//	              "type": "folder"
//	           }, {
//	              "date_added": "13152713865793130",
//	              "id": "15",
//	              "meta_info": {
//	                 "last_visited_desktop": "13152713865793354"
//	              },
//	              "name": "Homepage nl - Blogit",
//	              "type": "url",
//	              "url": "http://www.blogit.nl/"
//	           }, {
//	              "date_added": "13152970228641378",
//	              "id": "26",
//	              "meta_info": {
//	                 "last_visited_desktop": "13152970250760037"
//	              },
//	              "name": "Pointer free programming",
//	              "type": "url",
//	              "url": "https://nim-lang.org/araq/destructors.html"
//	           } ],
//	           "date_added": "13152713740466117",
//	           "date_modified": "13152970228641378",
//	           "id": "12",
//	           "name": "Blogs",
//	           "type": "folder"
//	        } ],
//	        "date_added": "13152359615589278",
//	        "date_modified": "13152713740466719",
//	        "id": "1",
//	        "name": "Bookmarks bar",
//	        "type": "folder"
//	     },
//	     "other": {
//	        "children": [  ],
//	        "date_added": "13152359615589283",
//	        "date_modified": "0",
//	        "id": "2",
//	        "name": "Other bookmarks",
//	        "type": "folder"
//	     }
//	  },
//	  "version": 1
//	}
package main

import (
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"
)

const QuantKnob = 5000

type Bookmark struct {
	DateAdded    string            `json:"date_added"`
	ID           string            `json:"id"`
	MetaInfo     map[string]string `json:"meta_info,omitempty"`
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	URL          string            `json:"url,omitempty"`
	Children     []Bookmark        `json:"children,omitempty"`
	DateModified string            `json:"date_modified,omitempty"`
}

func randomString(n int) string {
	letters := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func randomBookmark() Bookmark {
	bookmark := Bookmark{
		DateAdded: fmt.Sprintf("%d", time.Now().UnixNano()),
		ID:        fmt.Sprintf("%d", rand.Intn(10000)),
		Name:      randomString(10),
	}

	if rand.Float64() < 0.5 {
		bookmark.Type = "url"
		bookmark.URL = "https://" + randomString(10) + ".com"
	} else {
		bookmark.Type = "folder"
		var children []Bookmark
		for i := 0; i < rand.Intn(5); i++ {
			children = append(children, randomBookmark())
		}
		bookmark.Children = children
	}

	if rand.Float64() < 0.5 {
		bookmark.DateModified = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	if rand.Float64() < 0.5 {
		bookmark.MetaInfo = map[string]string{
			"last_visited_desktop": fmt.Sprintf("%d", time.Now().UnixNano()),
		}
	}

	return bookmark
}

func main() {

	var bookmarks []Bookmark

	bkAmount := flag.Int("amt", QuantKnob, "approximate amount of urls to generate")

	flag.Parse()

	for i := 0; i < *bkAmount; i++ {
		bookmarks = append(bookmarks, randomBookmark())
	}

	data := map[string]interface{}{
		"checksum": fmt.Sprintf("%x", md5.Sum([]byte(time.Now().String()))),
		"roots": map[string]interface{}{
			"bookmark_bar": Bookmark{
				DateAdded:    "13152359615589278",
				ID:           "1",
				Name:         "Bookmarks bar",
				Type:         "folder",
				Children:     bookmarks,
				DateModified: fmt.Sprintf("%d", time.Now().UnixNano()),
			},
			"other_bookmarks": Bookmark{
				DateAdded:    "13152359615589283",
				ID:           "2",
				Name:         "Other bookmarks",
				Type:         "folder",
				DateModified: "0",
			},
		},
		"version": 1,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", " ")
	enc.Encode(data)
}
