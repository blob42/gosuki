package firefox

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/chenhg5/collection"
	"github.com/stretchr/testify/assert"

	"github.com/blob42/gosuki/internal/database"
	"github.com/blob42/gosuki/internal/index"
	"github.com/blob42/gosuki/internal/logging"
	"github.com/blob42/gosuki/pkg/modules"
	"github.com/blob42/gosuki/pkg/browsers/mozilla"
	"github.com/blob42/gosuki/pkg/parsing"
	"github.com/blob42/gosuki/pkg/profiles"
	"github.com/blob42/gosuki/pkg/tree"
	"github.com/blob42/gosuki/internal/utils"
)

var ff Firefox

func TestMain(m *testing.M) {
	setupFirefox()

	exitVal := m.Run()
	os.Exit(exitVal)
}

func setupFirefox() {
	ff = Firefox{
		FirefoxConfig: &FirefoxConfig{
			BrowserConfig: &modules.BrowserConfig{
				Name:     "firefox",
				Type:     modules.TFirefox,
				BkFile:   mozilla.PlacesFile,
				BkDir:    "../../pkg/browsers/mozilla/testdata",
				BufferDB: &database.DB{},
				URLIndex: index.NewIndex(),
				NodeTree: &tree.Node{Name: mozilla.RootName, Parent: nil, Type: tree.RootNode},
				Stats:    &parsing.Stats{},
			},
		},
		tagMap:    map[string]*tree.Node{},
		folderMap: map[sqlid]*tree.Node{},
	}
}

func runPlacesTest(name string, t *testing.T, test func(t *testing.T)) {

	bkPath, err := ff.BookmarkPath()
	if err != nil {
		t.Error(err)
	}

	ff.places, err = database.NewDB("places", bkPath, database.DBTypeFileDSN,
		FFConfig.PlacesDSN).Init()

	if err != nil {
		t.Error(err)
	}

	defer func() {
		err = ff.places.Handle.Close()
		if err != nil {
			t.Error(err)
		}
		// Run the wal_checkpoint command to clean up the WAL file
		ff.places.Handle.Exec("PRAGMA wal_checkpoint(TRUNCATE)")

	}()

	t.Run(name, test)
}

func Test_addUrlNode(t *testing.T) {

	testURL := struct {
		url   string
		id    sqlid
		title string
		desc  string
	}{
		url:   "http://test-url.gosuki",
		id:    24,
		title: "test url",
		desc:  "desc of test url",
	}

	// fetch url changes into places and bookmarks
	// for each urlId/place
	// if urlNode does not exists create it
	// if urlNode exists find fetch it
	// if urlNode exists put tag node as parent to this url

	testNewURL := "new urlNode: url is not yet in URLIndex"

	t.Run(testNewURL, func(t *testing.T) {
		ok, urlNode := ff.addURLNode(testURL.url, testURL.title, testURL.desc)
		if !ok {
			t.Fatalf("expected %v, got %v", true, false)
		}
		if urlNode == nil {
			t.Fatal("url node was not returned", testNewURL)
		}

		_, ok = ff.URLIndex.Get(testURL.url)
		if !ok {
			t.Fatal("url was not added to url index")
		}

		if !utils.InList(ff.URLIndexList, testURL.url) {
			t.Fatal("url was not added to url index list")
		}

	})

	testURLExists := "return existing urlNode found in URLIndex"
	t.Run(testURLExists, func(t *testing.T) {
		_, origNode := ff.addURLNode(testURL.url, testURL.title, testURL.desc)
		ok, urlNode := ff.addURLNode(testURL.url, testURL.title, testURL.desc)
		if ok {
			t.Fatalf("expected %v, got %v", false, true)
		}

		if urlNode == nil {
			t.Fatal("existing url node was not returned from index")
		}

		if urlNode != origNode {
			t.Fatal("existing node does not match retrieved node from url index")
		}

		_, ok = ff.URLIndex.Get(testURL.url)
		if !ok {
			t.Fatal("url was not added to url index")
		}

		if !utils.InList(ff.URLIndexList, testURL.url) {
			t.Fatal("url was not added to url index list")
		}

	})

}

func Test_addFolderNode(t *testing.T) {

	// Test cases
	// 1. Adding a new folder under a root mozilla folder (parent = 2,3,5,6)
	// 2. Adding a child folder
	// 3. Adding a folder that we already saw before

	t.Run("adding firefox root folder", func(t *testing.T) {
		testRootFolder := MozFolder{
			Id:     3,
			Parent: 1,
			Title:  "toolbar",
		}

		created, fNode := ff.addFolderNode(testRootFolder)

		assert.True(t, created)

		// root folder should have appropriate title
		assert.Equal(t, fNode.Name, mozilla.RootFolderNames[mozilla.ToolbarID])

		// Should be underneath root folder
		assert.Equal(t, fNode.Parent, ff.NodeTree)

	})

	t.Run("add non existing folder with no parent", func(t *testing.T) {
		testFolder := MozFolder{
			Id:     10,
			Parent: 3, // folder under the Bookmarks Toolbar
			Title:  "Programming",
		}

		folderNodeCreated, folderNode := ff.addFolderNode(testFolder)

		// we should have the following hierarchy
		// -- ROOT
		//  |-- Bookmarks Toolbar
		//   |-- Programming

		// We expect the folder was created
		assert.True(t, folderNodeCreated)

		// If we add the same folder, we should get the same node from
		// the folderMap but no new folderNode is created
		folderAdded, sameFolderNode := ff.addFolderNode(testFolder)
		assert.False(t, folderAdded)
		assert.Equal(t, sameFolderNode, folderNode)

		assert.NotNil(t, folderNode, "folder was not created")

		// Folder should not be added at the root of the tree
		assert.NotEqual(t, folderNode.Parent, ff.NodeTree, "wront parent folder")

		// Name of node should match title of scanned folder
		assert.Equal(t, folderNode.Name, testFolder.Title, "parsing folder name")

        // If we add the same folder with differnt title it should update the folder name
        renamedFolder := testFolder
        renamedFolder.Title = "Dev"
        folderAdded, renamedFolderNode := ff.addFolderNode(renamedFolder)
        assert.Equal(t, folderNode, renamedFolderNode) // same folder node
        assert.False(t, folderAdded) // folder node is not created again
        assert.NotEqual(t, folderNode.Name, testFolder.Title)

	})
}

func Test_addTagNode(t *testing.T) {
    setupFirefox()

	testTag := struct {
		tagName string
		tagType string
	}{
		tagName: "#test_tag",
		tagType: "tag",
	}

	// Should return true with the new node
	testName := "add new tag to root tree"
	t.Run(testName, func(t *testing.T) {
		ok, tagNode := ff.addTagNode(testTag.tagName)
		if !ok {
			t.Errorf("[%s] expected %v ,got %v", testName, true, false)
		}
		if tagNode == nil {
			t.Fatalf("[%s] tag node was not returned", testName)
		}

		// "tags" branch should exist

		// TagNode should be underneath "tags" branch
		if tagNode.Parent.Parent != ff.NodeTree &&
			tagNode.Name != "tags" {
			t.Errorf("[%s] wrong parent root for tag", testName)
		}
		t.Run("should be in tagMap", func(t *testing.T) {
			node, ok := ff.tagMap[testTag.tagName]
			if !ok {
				t.Error("tag node was not found in tagMap")
			}

			if node != tagNode {
				t.Error("tag node different from the one added to tagMap")
			}
		})

		t.Run("increment node count", func(t *testing.T) {
			if ff.CurrentNodeCount != 1 {
				t.Errorf("wrong node count")
			}
		})
	})

	// This should return false with the existing node and not add a new one
	testName = "add existing tag to root tree"
	t.Run(testName, func(t *testing.T) {
		ff.addTagNode(testTag.tagName)
		ok, tagNode := ff.addTagNode(testTag.tagName)
		if tagNode == nil {
			t.Fatalf("[%s] tag node was not returned", testName)
		}
		if tagNode.Parent.Name != TagsBranchName {
			t.Errorf("[%s] wrong parent root for tag", testName)
		}
		if ok {
			t.Errorf("[%s] expected %v ,got %v", testName, false, true)
		}
	})
}

func Test_PlaceBookmarkTimeParsing(t *testing.T) {
	assert := assert.New(t)
	pb := mozilla.MergedPlaceBookmark{
		BkLastModified: 1663878015759000,
	}

	res := pb.Datetime().Format("2006-01-02 15:04:05.000000")
	assert.Equal(res, "2022-09-22 20:20:15.759000", "wrong time in scanned bookmark")
}

func findTagsInNodeTree(urlNode *tree.Node,
                        tags []string, // tags to find in tagMap
                        tagMap map[string]*tree.Node) (bool ,error) {
	var foundTagNodeForURL bool
	for _, tagName := range tags {
		tagNode, tagNodeExists := ff.tagMap[tagName]
		if !tagNodeExists {
			return false, fmt.Errorf("missing tag <%s>", tagName)
		}
		// Check that the URL node is a direct child of the tag node
		if urlNode.DirectChildOf(tagNode) {
			foundTagNodeForURL = true
		}
	}

    return foundTagNodeForURL, nil
}

// TODO!: integration test loading firefox bookmarks
func Test_scanBookmarks(t *testing.T) {
	logging.SetMode(-1)

	// expected data from testdata/places.sqlite
	data := struct {
		tags    []string
		folders []string // list of tags

		bookmarkTags map[string][]string // list of folder names

	}{ // list of urls which are bookmarked
		tags: []string{"golang", "programming", "rust"},

		folders: []string{
			"menu",    // Bookmarks Menu
			"toolbar", // Bookmarks Toolbar
			"tags",    // Tags Virtual Folder
			"unfiled", // Other Bookmarks
			"mobile",  // Mobile Bookmarks
			"cooking",
			"Travel",
			"indian",
			"GosukiMenu",
		},

		bookmarkTags: map[string][]string{
			"https://based.cooking/":                              {"based"},
			"https://go.dev/":                                     {"golang", "programming"},
			"https://www.rust-lang.org/":                          {"programming", "rust", "systems"},
			"https://www.tasteofhome.com/article/indian-cooking/": {},
			"http://gosuki.io/":                                   {"gosuki"},
			"https://www.budapestinfo.hu/":                        {"budapest"},
			"https://www.fsf.org/":                                {"libre"},
		},
	}

	t.Log("loading firefox bookmarks")

	// First make sure bookmarks are scaned then verify they are loaded
	// in CacheDB

	runPlacesTest("find", t, func(t *testing.T) {
		bookmarks, err := ff.scanBookmarks()
		if err != nil {
			t.Error(err)
		}

		// 1- find all tags defined by user
		t.Run("all urls", func(t *testing.T) {
			var urls []string
			for _, bk := range bookmarks {
				urls = utils.Extends(urls, bk.Url)
			}

			var testUrls []string
			for url := range data.bookmarkTags {
				testUrls = append(testUrls, url)
			}
			testUrls = collection.Collect(testUrls).Unique().ToStringArray()

			assert.ElementsMatch(t, urls, testUrls)


		})

		/*
		   2.find all folders
		*/
		t.Run("all folders", func(t *testing.T) {
			var folders []string
			for _, bk := range bookmarks {
				folderS := strings.Split(bk.Folders, ",")
				for _, f := range folderS {
					folders = utils.Extends(folders, f)
				}
			}
			assert.ElementsMatch(t, folders, data.folders)
		})

		/*
		   3. find all url bookmarks with their corresponding tags
		   - should get any user added bookmark (id > 12)
		*/
		t.Run("all tags", func(t *testing.T) {
			bkTags := map[string][]string{}

			for _, bk := range bookmarks {
				bkTags[bk.Url] = collection.Collect(strings.Split(bk.Tags, ",")).
					Unique().Filter(func(_, val interface{}) bool {
					// Filter out empty ("") strings
					if v, ok := val.(string); ok {
						if v == "" {
							return false
						}
					}
					return true
				}).ToStringArray()
			}

			assert.Equal(t, data.bookmarkTags, bkTags)
			// t.Error("urls with their matching tags")

			t.Run("should find all bookmarks that have tags AND within folders", func(t *testing.T) {
				for _, bk := range bookmarks {
					if bk.Url == "https://www.fsf.org/" {
						// should have `libre` tag and `Mobile Bookmarks` folder
						assert.Equal(t, bk.ParentFolder, "mobile")
					}
				}
			})

		})
	})

	runPlacesTest("load bookmarks in node tree", t, func(t *testing.T) {
		bookmarks, err := ff.scanBookmarks()
		if err != nil {
			t.Error(err)
		}
        ff.loadBookmarksToTree(bookmarks)

		t.Run("find every url in the node tree", func(t *testing.T) {

			for _, bk := range bookmarks {
				node, exists := ff.URLIndex.Get(bk.Url)
				assert.True(t, exists, "url missing in URLIndex")

				assert.True(t, tree.FindNode(node.(*tree.Node), ff.NodeTree), "url node missing from tree")
			}
		})

		t.Run("url node is child of the right tag nodes", func(t *testing.T) {
			// Every URL node should be a child of the right tag node

			// Go through each tag node
			for _, bk := range bookmarks {

				urlNode, urlNodeExists := ff.URLIndex.Get(bk.Url)
				assert.True(t, urlNodeExists, "url missing in URLIndex")

				// only check bookmarks with tags
				if len(bk.Tags) == 0 {
					continue
				}

                tags := strings.Split(bk.Tags, ",")
                foundTagNodeForUrl, err := findTagsInNodeTree(urlNode.(*tree.Node),
                                                              tags, ff.tagMap)
                if err != nil {
                  t.Error(err)
                }
                

				assert.True(t, foundTagNodeForUrl)

			}
		})

		t.Run("url underneath the right folders", func(t *testing.T) {
			for _, bk := range bookmarks {
				// folder, folderScanned := ff.folderScanMap[bk.ParentId]
				//  assert.True(t, folderScanned)

				// Get the folder from tree node
				folderNode, folderExists := ff.folderMap[bk.ParentId]
				assert.True(t, folderExists)

				urlNode, exists := ff.URLIndex.Get(bk.Url)
				assert.True(t, exists, "url missing in URLIndex")

				// check that url node has the right parent folder node

				// If Parent is nil, it means no folder was assigned to this url node
				parentFolder := bk.ParentFolder
				switch parentFolder {
				case "unfiled":
					parentFolder = mozilla.RootFolderNames[mozilla.OtherID]
				case "mobile":
					parentFolder = mozilla.RootFolderNames[mozilla.MobileID]
				}
				if urlNode.(*tree.Node).Parent != nil {
					assert.Equal(t, parentFolder, urlNode.(*tree.Node).Parent.Name, 
						"wrong folder for <%s>", bk.Url)
				}

				assert.True(t, urlNode.(*tree.Node).DirectChildOf(folderNode),
					"missing folder for %s", bk.Url)

			}
		})
	})
}

func Test_scanFolders(t *testing.T) {

	folders := []string{
		"menu",    // Bookmarks Menu
		"toolbar", // Bookmarks Toolbar
		"tags",    // Tags Virtual Folder
		"unfiled", // Other Bookmarks
		"mobile",  // Mobile Bookmarks
		"Mozilla Firefox",
		"cooking",
		"Travel",
		"indian",
		"GosukiMenu",
	}

	runPlacesTest("scan all folders", t, func(t *testing.T) {

		// query all folders
		scannedFolders, err := ff.scanFolders(0)
		if err != nil {
			t.Error(err)
		}

		// test that we loaded all folders
		folderS := []string{}
		for _, f := range scannedFolders {
			folderS = utils.Extends(folderS, f.Title)
		}
		assert.ElementsMatch(t, folders, folderS)

		// testing the tree

		// folderMap should have 9 entries (id=4 is reserved for tags)
		assert.Equal(t, len(ff.folderMap), 9, "not all nodes present in folderMap")

		// test that folders are loaded into tree
		// All folders can reach the root ancestor
		for _, f := range ff.folderMap {
			assert.Equal(t, ff.NodeTree, tree.Ancestor(f), "all folders attached to root")

			//every folder in folderMap has a corresponding node in the tree
			assert.True(t, tree.FindNode(f, ff.NodeTree), "folder nodes are attached to tree")
		}

	})

}

func Test_FindModifiedBookmarks(t *testing.T) {
	//NOTE: use a separate test places db that includes changes vs the main test db
	// Test scenarios
	// 1. Modify an existing bookmark
	//  a. Add / Remove tag ( find new tags )
	//  b. Move to folder ( find new folder)
	//  TODO: c. DELETE bookmark
	// 2. Find new bookmarks
	// 2. Find new created tags
	// 3. Find new created folders (even if empty)

    type bkTestData struct {
        tags []string
        folders []string
    }

    modifiedBookmarks := map[string]bkTestData{
        "https://go.dev/": {
            tags:    []string{"language"},
            folders: []string{mozilla.RootFolderNames[mozilla.OtherID]}, // unfiled folder
        },
    }


    newBookmarks := map[string]bkTestData{
        "https://bitcoinwhitepaper.co/": {
        tags:    []string{"bitcoin"},
        folders: []string{
            "Cryptocurrencies",
            mozilla.RootFolderNames[mozilla.OtherID],
            mozilla.RootFolderNames[mozilla.ToolbarID],
        },

        },
        "https://lightning.network/": {
        tags:    []string{"bitcoin", "lightning"},
        folders: []string{mozilla.RootFolderNames[mozilla.OtherID]},
        },
    }

    newFolders := []string{"Cryptocurrencies", "NewFolder"}

    //TODO!: modified folders

	// Setup the appropriate test db
	ff.BkFile = "places-modified.sqlite"

	runPlacesTest("loading changes to node tree", t, func(t *testing.T) {
		scanModifiedSince := 1672688367910000
		bookmarks, err := ff.scanModifiedBookmarks(int64(scanModifiedSince))
		if err != nil {
			t.Error(err)
		}
        ff.loadBookmarksToTree(bookmarks)

        t.Run("modified bookmarks", func(t *testing.T){
            // test that each modified bookmark is is loaded
            // in the node tree
            for modUrl, modBk := range modifiedBookmarks {
                node, exists := ff.URLIndex.Get(modUrl)      
				assert.True(t, exists, "url missing in URLIndex")
                urlNode := node.(*tree.Node)

				assert.True(t, tree.FindNode(node.(*tree.Node), ff.NodeTree), "url node missing from tree")


                // matching tag nodes
                found, err := findTagsInNodeTree(node.(*tree.Node), modBk.tags, ff.tagMap)
                if err != nil {
                  t.Error(err)
                }
                
                assert.True(t, found, "missing tag")


                // matching folders
                parentFolders := tree.FindParents(ff.NodeTree, urlNode, tree.FolderNode)
                pFolderNames := utils.Map(func(node *tree.Node) string{
                    return node.Name
                }, parentFolders)

                assert.ElementsMatch(t, pFolderNames, modBk.folders)


            }
        })

        t.Run("new bookmarks", func(t *testing.T){
            for newURL, newBk := range newBookmarks {

                node, exists := ff.URLIndex.Get(newURL)      
                urlNode := node.(*tree.Node)
				assert.True(t, exists, "url missing in URLIndex")
				assert.True(t, tree.FindNode(urlNode, ff.NodeTree), "url node missing from tree")

                // matching tag nodes
                found, err := findTagsInNodeTree(urlNode, newBk.tags, ff.tagMap)
                if err != nil {
                  t.Error(err)
                }

                assert.True(t, found, "missing tag")


                parentFolders := tree.FindParents(ff.NodeTree, urlNode, tree.FolderNode)
                pFolderNames := utils.Map(func(node *tree.Node) string{
                    return node.Name
                }, parentFolders)

                assert.ElementsMatch(t, pFolderNames, newBk.folders, fmt.Sprintf("mismatch folder for <%s>", urlNode.URL))
            }

        })

        t.Run("find new folders", func(t *testing.T){
            // Make sure the new folders exist in the folder node map
            var folderNodes []*tree.Node
            for _, fNode := range ff.folderMap {
                folderNodes = append(folderNodes, fNode)
            }

            // Get all folder names in tree
            folderNames := utils.Map(func(node *tree.Node) string{
                return node.Name
            }, folderNodes)

            assert.Subset(t, folderNames, newFolders)
        })
	})
}

func TestBrowserImplProfileManager(t *testing.T) {
	assert.Implements(t, (*profiles.ProfileManager)(nil), NewFirefox())
}


func Test_FindModifiedFolders(t *testing.T) {
   t.Skip("modified folder names should change the corresponding bookmark tags") 
}
