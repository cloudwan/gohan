// Copyright (C) 2017 NTT Innovation Institute, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hash

import (
	"sort"
)

// Run compresses items that have the same hash and level th parent
func Run(item IHashable, level int) {
	tree := createTree(item)
	tree.calcHashes()
	tree.compress(level)
}

type treeNode struct {
	index     int
	item      IHashable
	value     Node
	hash      *Hash
	ancestor  *treeNode
	ancestors []*treeNode
	children  []*treeNode
}

func createTree(item IHashable) *treeNode {
	var (
		index          int
		createTreeNode func(IHashable, *treeNode, *Hash) *treeNode
	)

	createTreeNode = func(item IHashable, parent *treeNode, hash *Hash) *treeNode {
		children := item.GetChildren()
		tree := &treeNode{
			index:     index,
			item:      item,
			ancestors: []*treeNode{parent},
			hash:      hash,
			children:  make([]*treeNode, len(children)),
		}
		index++
		for i, child := range children {
			tree.children[i] = createTreeNode(child, tree, hash)
		}
		return tree
	}

	return createTreeNode(item, nil, &Hash{})
}

func (tree *treeNode) allNodes() []*treeNode {
	var getNodes func(*treeNode)

	result := []*treeNode{}

	getNodes = func(node *treeNode) {
		result = append(result, node)
		for _, child := range node.children {
			getNodes(child)
		}
	}

	getNodes(tree)
	return result
}

func (tree *treeNode) calcHashes() {
	joinHashes := func(hash *Hash, item IHashable, nodes []Node) Node {
		result := hash.Calc(item.ToString() + "(")
		for _, node := range nodes {
			result = hash.Join(result, node)
			result = hash.Join(result, hash.Calc(","))
		}
		return hash.Join(result, hash.Calc(")"))
	}

	nodes := make([]Node, len(tree.children))
	for i, node := range tree.children {
		node.calcHashes()
		nodes[i] = node.value
	}
	tree.value = joinHashes(tree.hash, tree.item, nodes)
}

func (tree *treeNode) compress(level int) {
	if level < 1 {
		return
	}

	getAncestors := func(nodes []*treeNode) {
		log, powers := powers(level)
		allNil := len(nodes) <= 1

		for i := 0; i < log && !allNil; i++ {
			allNil = true
			for _, node := range nodes {
				if ancestor := node.ancestors[i]; ancestor != nil {
					allNil = false
					node.ancestors = append(node.ancestors, ancestor.ancestors[i])
				} else {
					node.ancestors = append(node.ancestors, nil)
				}
			}
		}

		if len(nodes[0].ancestors) < log+1 {
			return
		}

		for _, node := range nodes {
			node.ancestor = node
			for _, value := range powers {
				if node.ancestor == nil {
					break
				}
				node.ancestor = node.ancestor.ancestors[value]
			}
		}
	}

	compressNodes := func(nodes []*treeNode) {
		sort.Sort(byHash(nodes))

		var index int
		for index = 0; index < len(nodes) && nodes[index].ancestor == nil; index++ {
		}
		index++
		for ; index < len(nodes); index++ {
			if nodes[index].value == nodes[index-1].value &&
				nodes[index].ancestor == nodes[index-1].ancestor {
				nodes[index].ancestors[0].item.Compress(nodes[index-1].item, nodes[index].item)
				nodes[index].item = nodes[index-1].item
			}
		}
	}

	nodes := tree.allNodes()
	getAncestors(nodes)
	compressNodes(nodes)
}

// Sorting nodes by hash
type byHash []*treeNode

func (array byHash) Len() int {
	return len(array)
}

func (array byHash) Swap(i, j int) {
	array[i], array[j] = array[j], array[i]
}

func (array byHash) Less(i, j int) bool {
	if array[i].ancestor == nil {
		return true
	}
	if array[j].ancestor == nil {
		return false
	}
	if array[i].value.value == array[j].value.value {
		if array[i].value.length == array[j].value.length {
			return array[i].ancestor.index < array[j].ancestor.index
		}
		return array[i].value.length < array[j].value.length
	}
	return array[i].value.value < array[j].value.value

}
