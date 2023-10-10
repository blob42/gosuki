//
// Copyright ⓒ 2023 Chakib Ben Ziane <contact@blob42.xyz> and [`GoSuki` contributors]
// (https://github.com/blob42/gosuki/graphs/contributors).
//
// All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This file is part of GoSuki.
//
// GoSuki is free software: you can redistribute it and/or modify it under the terms of
// the GNU Affero General Public License as published by the Free Software Foundation,
// either version 3 of the License, or (at your option) any later version.
//
// GoSuki is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
// without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR
// PURPOSE.  See the GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License along with
// gosuki.  If not, see <http://www.gnu.org/licenses/>.

package tree

import (
	"fmt"

	"github.com/blob42/gosuki/pkg/bookmarks"
	"github.com/blob42/gosuki/internal/index"
	"github.com/blob42/gosuki/internal/logging"
	"github.com/blob42/gosuki/internal/utils"
	"github.com/kr/pretty"

	"github.com/xlab/treeprint"
)

var log = logging.GetLogger("TREE")


type NodeType int

const (
	RootNode NodeType = iota
	URLNode
	FolderNode
	TagNode
)

type Node struct {
	Name       string
	Type       NodeType // folder, tag, url
	URL        string
	Tags       []string
	Desc       string
	HasChanged bool
	NameHash   uint64 // hash of the metadata
	Parent     *Node
	Children   []*Node
}


func (node *Node) GetRoot() *Node {
    var n *Node
    if node.Type == RootNode {
        return node
    } 

    if node.Parent != nil  {
        n = node.Parent.GetRoot()
    }
    return n
}

// Returns the ancestor of this node
func Ancestor(node *Node) *Node {
	if node.Parent == nil {
		return node
	} else {
		return Ancestor(node.Parent)
	}
}

func (node *Node) DirectChildOf(parent *Node) bool {
    if len(parent.Children) == 0 { return false }
    var found bool
    for _, child := range parent.Children {
        if node == child { found = true }
    }

    return found
}

// Get all parents for node by traversing from leaf to root
func (node *Node) GetFolderParents() []*Node {
    var parents []*Node

    if node.Parent == nil {
        return parents
    }

    if node.Parent.Type == FolderNode {
        parents = append(parents, node.Parent)
    }

    if node.Parent.Type != RootNode {
        parents = append(parents, node.Parent.GetFolderParents()...)
    }

    return parents
}

// Recursively traverse the tree from a root and find all occurences of [url]
// whose parent is a folder without using url.Parent as a reference
// Returns a list of nodes that match the criteria
func FindParents(root *Node, url *Node, nt NodeType) []*Node {
    var parents []*Node

    if root == nil || len(root.Children) <= 0 {
        return parents
    }

    if root.Type == nt && FindNode(url, root) {
        parents = append(parents, root)
    }

    for _, child := range root.Children {
        parents = append(parents, FindParents(child, url, nt)...)
    }

    return parents
}


// Finds a node and the tree starting at root
func FindNode(node *Node, root *Node) bool {

	if node == root {
		return true
	} else {
        for _, child := range root.Children {
            found := FindNode(node, child)
            if found { return true }
        }
    }

    return false
}

func FindNodeByName(name string, root *Node) bool {

	if name == root.Name {
		return true
	} else {
        for _, child := range root.Children {
            found := FindNodeByName(name, child)
            if found { return true }
        }
    }

    return false
}


// Inserts child node into parent node. Parent will point to child
// and child will point to parent EXCEPT when parent is a TAG node.
// If parent is a Tag node, child should not point back to parent
// as URL nodes should always point to folder parent nodes only.
func AddChild(parent *Node, child *Node) {
	log.Debugf("adding child %v: <%s>", child.Type, child.Name)


	if len(parent.Children) == 0 {
		parent.Children = []*Node{child}

        // Do not point back to TAG parent node from child
        if parent.Type != TagNode {
            child.Parent = parent
        }
		return
	}

	for _, n := range parent.Children {
		if child == n {
			// log.Errorf("<%s> Node already exists", child)
            log.Debugf(pretty.Sprintf("skipping node <%s>, already exists", child.Name))
			// update n with child metadata
			log.Debugf("updating node <%s> with metadata <%s>", n.Name, child.Name)
			n.Name = child.Name
			return
		}
	}

	parent.Children = append(parent.Children, child)
    if parent.Type != TagNode {
        child.Parent = parent
    }
}

func PrintTree(root *Node) {
	fmt.Println("---")
	fmt.Println("PrintTree")
	var walk func(node *Node, tree treeprint.Tree)
	tree := treeprint.New()

	walk = func(node *Node, t treeprint.Tree) {

		if len(node.Children) > 0 {
			t = t.AddBranch(fmt.Sprintf("%#v <%s>", node.Type, node.Name))

			for _, child := range node.Children {
				walk(child, t)
			}
		} else {
			t.AddNode(fmt.Sprintf("%#v <%s>", node.Type, node.Name))
		}
	}

	walk(root, tree)
	fmt.Println(tree.String())
	fmt.Println("---")
}

// Rebuilds the memory url index after parsing all bookmarks.
// Keeps memory index in sync with last known state of browser bookmarks
func WalkBuildIndex(node *Node, index index.HashTree) {
	if node.Type == URLNode {
		index.Insert(node.URL, node)
		//log.Debugf("Inserted URL: %s and Hash: %v", node.URL, node.NameHash)
	}

	if len(node.Children) > 0 {
		for _, node := range node.Children {
			WalkBuildIndex(node, index)
		}

	}
}

// Get all possible tags for this url node The tags make sense only in the
// context of a URL node This will traverse the three breadth first to find all
// parent folders and add them as a tag. URL nodes should already be populated
// with the list of tags that exist under the TAG tree. So we only need to find
// the parent folders and turn them into tags.
func (node *Node) getTags() []string {

    if node.Type != URLNode {
        return []string{}
    }

    root := node.GetRoot()
    parentFolders := FindParents(root, node, FolderNode)
    parentTags := FindParents(root, node, TagNode)

    for _, f := range parentFolders {
        node.Tags = utils.Extends(node.Tags, f.Name)
    }

    for _, t := range parentTags {
        node.Tags = utils.Extends(node.Tags, t.Name)
    }

    return node.Tags
}

func (node *Node) GetBookmark() *bookmarks.Bookmark {

    if node.Type != URLNode {
        return nil
    }

	return &bookmarks.Bookmark{
		URL:      node.URL,
		Metadata: node.Name,
		Desc:     node.Desc,
		Tags:     node.getTags(),
	}
}
