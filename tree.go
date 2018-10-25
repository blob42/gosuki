package main

import (
	"fmt"

	"github.com/xlab/treeprint"
)

type Node struct {
	Name       string
	Type       string // folder, tag, url
	URL        string
	Tags       []string
	Desc       string
	HasChanged bool
	NameHash   uint64 // hash of the metadata
	Parent     *Node
	Children   []*Node
}

func (node *Node) GetBookmark() *Bookmark {
	return &Bookmark{
		URL:      node.URL,
		Metadata: node.Name,
		Desc:     node.Desc,
		Tags:     node.Tags,
		Node:     node,
	}
}

func (node *Node) GetRoot() *Node {
	nodePtr := node

	for {
		if nodePtr.Name == "root" {
			break
		}

		nodePtr = nodePtr.Parent
	}

	return nodePtr
}

// Returns all parent tags for URL nodes
func (node *Node) GetParentTags() []*Node {
	var parents []*Node
	var walk func(node *Node)
	var nodePtr *Node

	root := node.GetRoot()

	walk = func(n *Node) {
		nodePtr = n

		//log.Debugf("type of %s --> %s", nodePtr.Type, nodePtr.Name)
		if nodePtr.Type == "url" {
			return
		}

		if len(nodePtr.Children) == 0 {
			return
		}

		for _, v := range nodePtr.Children {
			if v.URL == node.URL &&
				nodePtr.Type == "tag" {
				parents = append(parents, nodePtr)
			}
			walk(v)
		}
	}

	walk(root)
	return parents
}

func PrintTree(root *Node) {
	var walk func(node *Node, tree treeprint.Tree)
	tree := treeprint.New()

	walk = func(node *Node, t treeprint.Tree) {

		if len(node.Children) > 0 {
			t = t.AddBranch(fmt.Sprintf("%s <%s>", node.Type, node.Name))

			for _, child := range node.Children {
				walk(child, t)
			}
		} else {
			t.AddNode(fmt.Sprintf("%s <%s>", node.Type, node.URL))
		}
	}

	walk(root, tree)
	log.Debug(tree.String())

}

// Debuggin bookmark node tree
// TODO: Better usage of node trees
func WalkNode(node *Node) {
	if node.Name == "root" {
		log.Debugf("Node --> <name: %s> | <type: %s> | children: %d | parent: %v", node.Name, node.Type, len(node.Children), node.Name)
	} else {
		log.Debugf("Node --> <name: %s> | <type: %s> | children: %d | parent: %v", node.Name, node.Type, len(node.Children), node.Parent.Name)
	}

	if len(node.Children) > 0 {
		for _, node := range node.Children {
			go WalkNode(node)
		}
	}
}

// Rebuilds the memory url index after parsing all bookmarks.
// Keeps memory index in sync with last known state of browser bookmarks
func WalkBuildIndex(node *Node, b *BaseBrowser) {

	if node.Type == "url" {
		b.URLIndex.Insert(node.URL, node)
		//log.Debugf("Inserted URL: %s and Hash: %v", node.URL, node.NameHash)
	}

	if len(node.Children) > 0 {
		for _, node := range node.Children {
			go WalkBuildIndex(node, b)
		}

	}
}

func syncTreeToBuffer(node *Node, buffer *DB) {

	if node.Type == "url" {
		bk := node.GetBookmark()
		bk.InsertOrUpdateInDB(buffer)
	}

	if len(node.Children) > 0 {
		for _, node := range node.Children {
			syncTreeToBuffer(node, buffer)
		}
	}
}
