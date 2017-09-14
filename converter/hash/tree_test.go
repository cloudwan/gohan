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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type testNode struct {
	value string
	nodes []*testNode
}

func (node *testNode) ToString() string {
	return node.value
}

func (node *testNode) Compress(source IHashable, destination IHashable) {
	for i, elem := range node.nodes {
		if destination.(*testNode) == elem {
			node.nodes[i] = source.(*testNode)
			break
		}
	}
}

func (node *testNode) GetChildren() []IHashable {
	result := make([]IHashable, len(node.nodes))
	for i, elem := range node.nodes {
		result[i] = elem
	}
	return result
}

var _ = Describe("tree tests", func() {
	var (
		onlyRoot = func() *testNode {
			return &testNode{
				"root",
				nil,
			}
		}

		createSmall = func() *testNode {
			return &testNode{
				"root",
				[]*testNode{
					{
						"a",
						nil,
					},
					{
						"a",
						nil,
					},
				},
			}
		}

		createSmallMiddleHeight = func() *testNode {
			return &testNode{
				"root",
				[]*testNode{
					{
						"a",
						[]*testNode{
							{
								"a",
								nil,
							},
						},
					},
					{
						"a",
						[]*testNode{
							{
								"a",
								nil,
							},
						},
					},
				},
			}
		}

		createSmallHigh = func() *testNode {
			return &testNode{
				"root",
				[]*testNode{
					{
						"a",
						[]*testNode{
							{
								"a",
								[]*testNode{
									{
										"a",
										[]*testNode{
											{
												"a",
												nil,
											},
										},
									},
								},
							},
						},
					},
					{
						"a",
						[]*testNode{
							{
								"a",
								[]*testNode{
									{
										"a",
										[]*testNode{
											{
												"a",
												nil,
											},
										},
									},
								},
							},
						},
					},
				},
			}
		}

		createWide = func() *testNode {
			return &testNode{
				"root",
				[]*testNode{
					{
						"a",
						[]*testNode{
							{
								"a",
								[]*testNode{
									{
										"b",
										nil,
									},
								},
							},
							{
								"a",
								[]*testNode{
									{
										"b",
										nil,
									},
								},
							},
						},
					},
					{
						"b",
						[]*testNode{
							{
								"a",
								[]*testNode{
									{
										"b",
										nil,
									},
								},
							},
							{
								"a",
								[]*testNode{
									{
										"b",
										nil,
									},
								},
							},
						},
					},
					{
						"a",
						[]*testNode{
							{
								"a",
								[]*testNode{
									{
										"b",
										nil,
									},
								},
							},
							{
								"a",
								[]*testNode{
									{
										"b",
										nil,
									},
								},
							},
						},
					},
				},
			}
		}
	)

	Describe("create tree tests", func() {
		It("Should create a root only tree", func() {
			item := onlyRoot()

			tree := createTree(item)

			Expect(tree.index).To(Equal(0))
			Expect(tree.item).To(Equal(item))
			Expect(len(tree.ancestors)).To(Equal(1))
			Expect(tree.ancestors[0]).To(BeNil())
			Expect(len(tree.children)).To(Equal(0))
		})

		It("Should create a small tree", func() {
			item := createSmall()

			tree := createTree(item)

			Expect(tree.index).To(Equal(0))
			Expect(tree.item).To(BeIdenticalTo(item))
			Expect(len(tree.ancestors)).To(Equal(1))
			Expect(tree.ancestors[0]).To(BeNil())
			Expect(len(tree.children)).To(Equal(2))

			hash := tree.hash
			left := tree.children[0]

			Expect(left.hash).To(BeIdenticalTo(hash))
			Expect(left.index).To(Equal(1))
			Expect(left.item).To(BeIdenticalTo(item.nodes[0]))
			Expect(len(left.ancestors)).To(Equal(1))
			Expect(left.ancestors[0]).To(BeIdenticalTo(tree))
			Expect(len(left.children)).To(Equal(0))

			right := tree.children[1]

			Expect(right.hash).To(BeIdenticalTo(hash))
			Expect(right.index).To(Equal(2))
			Expect(right.item).To(BeIdenticalTo(item.nodes[1]))
			Expect(len(right.ancestors)).To(Equal(1))
			Expect(right.ancestors[0]).To(BeIdenticalTo(tree))
			Expect(len(right.children)).To(Equal(0))
		})
	})

	Describe("all nodes tests", func() {
		It("Should get all nodes for a root only tree", func() {
			item := onlyRoot()
			tree := createTree(item)

			result := tree.allNodes()

			Expect(len(result)).To(Equal(1))
			Expect(result[0]).To(BeIdenticalTo(tree))
		})

		It("Should get all nodes for a small tree", func() {
			item := createSmall()
			tree := createTree(item)

			result := tree.allNodes()

			Expect(len(result)).To(Equal(3))
			Expect(result[0]).To(BeIdenticalTo(tree))
			Expect(result[1]).To(BeIdenticalTo(tree.children[0]))
			Expect(result[2]).To(BeIdenticalTo(tree.children[1]))
		})

		It("Should get all nodes for a small tree with middle height", func() {
			item := createSmallMiddleHeight()
			tree := createTree(item)

			result := tree.allNodes()

			Expect(len(result)).To(Equal(5))
			Expect(result[0]).To(BeIdenticalTo(tree))
			Expect(result[1]).To(BeIdenticalTo(tree.children[0]))
			Expect(result[2]).To(BeIdenticalTo(tree.children[0].children[0]))
			Expect(result[3]).To(BeIdenticalTo(tree.children[1]))
			Expect(result[4]).To(BeIdenticalTo(tree.children[1].children[0]))
		})
	})

	Describe("calc hash tests", func() {
		It("Should calc correct hashes for a root only tree", func() {
			item := onlyRoot()
			tree := createTree(item)

			tree.calcHashes()

			hash := tree.hash
			Expect(tree.value).To(Equal(hash.Calc("root()")))
		})

		It("Should calc correct hashes for a small tree", func() {
			item := createSmall()
			tree := createTree(item)

			tree.calcHashes()

			hash := tree.hash
			Expect(tree.value).To(Equal(hash.Calc("root(a(),a(),)")))
			Expect(tree.children[0].value).To(Equal(hash.Calc("a()")))
			Expect(tree.children[1].value).To(Equal(hash.Calc("a()")))
		})

		It("Should calc correct hashes for a small tree with middle height", func() {
			item := createSmallMiddleHeight()
			tree := createTree(item)

			tree.calcHashes()

			hash := tree.hash
			Expect(tree.value).To(Equal(hash.Calc("root(a(a(),),a(a(),),)")))
			Expect(tree.children[0].value).To(Equal(hash.Calc("a(a(),)")))
			Expect(tree.children[1].value).To(Equal(hash.Calc("a(a(),)")))
			Expect(tree.children[0].children[0].value).To(Equal(hash.Calc("a()")))
			Expect(tree.children[1].children[0].value).To(Equal(hash.Calc("a()")))

		})
	})

	Describe("compress tree tests", func() {
		Describe("ancestor tests", func() {
			It("Should get ancestors for a root only tree", func() {
				item := onlyRoot()
				tree := createTree(item)
				tree.calcHashes()

				tree.compress(2)

				Expect(len(tree.ancestors)).To(Equal(1))
				Expect(tree.ancestors[0]).To(BeNil())
				Expect(tree.ancestor).To(BeNil())
			})

			It("Should get ancestor for a high tree", func() {
				item := createSmallHigh()
				tree := createTree(item)
				tree.calcHashes()

				tree.compress(7)

				nodes := tree.allNodes()

				Expect(nodes[0].ancestors).To(Equal([]*treeNode{nil, nil, nil}))
				Expect(nodes[1].ancestors).To(Equal([]*treeNode{nodes[0], nil, nil}))
				Expect(nodes[2].ancestors).To(Equal([]*treeNode{nodes[1], nodes[0], nil}))
				Expect(nodes[3].ancestors).To(Equal([]*treeNode{nodes[2], nodes[1], nil}))
				Expect(nodes[4].ancestors).To(Equal([]*treeNode{nodes[3], nodes[2], nodes[0]}))
				Expect(nodes[5].ancestors).To(Equal([]*treeNode{nodes[0], nil, nil}))
				Expect(nodes[6].ancestors).To(Equal([]*treeNode{nodes[5], nodes[0], nil}))
				Expect(nodes[7].ancestors).To(Equal([]*treeNode{nodes[6], nodes[5], nil}))
				Expect(nodes[8].ancestors).To(Equal([]*treeNode{nodes[7], nodes[6], nodes[0]}))

				for _, node := range nodes {
					Expect(node.ancestor).To(BeNil())
				}
			})

			It("Should not overflow", func() {
				item := createSmallHigh()
				tree := createTree(item)
				tree.calcHashes()

				tree.compress(3)

				nodes := tree.allNodes()

				Expect(nodes[0].ancestors).To(Equal([]*treeNode{nil, nil}))
				Expect(nodes[1].ancestors).To(Equal([]*treeNode{nodes[0], nil}))
				Expect(nodes[2].ancestors).To(Equal([]*treeNode{nodes[1], nodes[0]}))
				Expect(nodes[3].ancestors).To(Equal([]*treeNode{nodes[2], nodes[1]}))
				Expect(nodes[4].ancestors).To(Equal([]*treeNode{nodes[3], nodes[2]}))
				Expect(nodes[5].ancestors).To(Equal([]*treeNode{nodes[0], nil}))
				Expect(nodes[6].ancestors).To(Equal([]*treeNode{nodes[5], nodes[0]}))
				Expect(nodes[7].ancestors).To(Equal([]*treeNode{nodes[6], nodes[5]}))
				Expect(nodes[8].ancestors).To(Equal([]*treeNode{nodes[7], nodes[6]}))

				for i, node := range nodes {
					switch i {
					case 3, 7:
						Expect(node.ancestor).To(BeIdenticalTo(nodes[0]))
					case 4:
						Expect(node.ancestor).To(BeIdenticalTo(nodes[1]))
					case 8:
						Expect(node.ancestor).To(BeIdenticalTo(nodes[5]))
					default:
						Expect(node.ancestor).To(BeNil())
					}
				}
			})

			It("Should skip non positive level", func() {
				item := createSmall()
				tree := createTree(item)
				tree.calcHashes()

				tree.compress(0)

				nodes := tree.allNodes()

				for _, node := range nodes {
					Expect(node.ancestor).To(BeNil())
				}
			})
		})

		Describe("compression tests", func() {
			type pair struct {
				first,
				second int
			}

			var (
				ok map[pair]bool

				createIdentical = func(identical [][]int) {
					for _, array := range identical {
						for i := range array {
							for j := i + 1; j < len(array); j++ {
								ok[pair{array[i], array[j]}] = true
							}
						}
					}
				}

				check = func(tree *treeNode) {
					nodes := tree.allNodes()
					for i := range nodes {
						for j := i + 1; j < len(nodes); j++ {
							if ok[pair{i, j}] {
								Expect(nodes[i].item).To(BeIdenticalTo(nodes[j].item))
							} else {
								Expect(nodes[i].item).ToNot(BeIdenticalTo(nodes[j].item))
							}
						}
					}
				}
			)

			BeforeEach(func() {
				ok = map[pair]bool{}
			})

			It("Should compress a wide tree level 1", func() {
				item := createWide()
				tree := createTree(item)
				tree.calcHashes()

				tree.compress(1)

				createIdentical([][]int{{1, 11}, {2, 4}, {7, 9}, {12, 14}})
				check(tree)
			})

			It("Should compress a wide tree level 3", func() {
				item := createWide()
				tree := createTree(item)
				tree.calcHashes()

				tree.compress(3)

				createIdentical([][]int{{3, 5, 8, 10, 13, 15}})
				check(tree)
			})

			It("Should compress a high tree level 4", func() {
				item := createSmallHigh()
				tree := createTree(item)
				tree.calcHashes()

				tree.compress(4)

				createIdentical([][]int{{4, 8}})
				check(tree)
			})
		})
	})

	Describe("general tests", func() {
		It("Should compress a wide tree level 1", func() {
			item := createWide()

			Run(item, 1)

			Expect(item.nodes[0]).To(BeIdenticalTo(item.nodes[2]))
			Expect(item.nodes[0]).ToNot(BeIdenticalTo(item.nodes[1]))
		})

		It("Should compress a wide tree level 3", func() {
			item := createWide()

			Run(item, 3)

			Expect(item.nodes[0].nodes[0].nodes[0]).To(
				BeIdenticalTo(item.nodes[0].nodes[1].nodes[0]),
			)
			Expect(item.nodes[0].nodes[1].nodes[0]).To(
				BeIdenticalTo(item.nodes[1].nodes[0].nodes[0]),
			)
			Expect(item.nodes[1].nodes[0].nodes[0]).To(
				BeIdenticalTo(item.nodes[1].nodes[1].nodes[0]),
			)
			Expect(item.nodes[1].nodes[1].nodes[0]).To(
				BeIdenticalTo(item.nodes[2].nodes[0].nodes[0]),
			)
			Expect(item.nodes[2].nodes[0].nodes[0]).To(
				BeIdenticalTo(item.nodes[2].nodes[1].nodes[0]),
			)
		})

		It("Should compress a high tree level 4", func() {
			item := createSmallHigh()

			Run(item, 4)

			Expect(item.nodes[0].nodes[0].nodes[0].nodes[0]).To(
				BeIdenticalTo(item.nodes[1].nodes[0].nodes[0].nodes[0]),
			)
			Expect(item.nodes[0].nodes[0].nodes[0]).ToNot(
				BeIdenticalTo(item.nodes[1].nodes[0].nodes[0]),
			)
		})
	})
})
