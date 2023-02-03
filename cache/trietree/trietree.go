package trietree

import (
	"regexp"
	"strings"
)

// TrieTree 前缀树
type TrieTree struct {
	replaceChar []rune
	root        *TrieNode
}

type TrieNode struct {
	ChildMap map[rune]*TrieNode // 本节点下的所有子节点
	Data     string             // 在最后一个节点保存完整的一个内容
	End      bool               // 标识是否最后一个节点
}

func NewTrieTree(replaceChar string) *TrieTree {
	return &TrieTree{
		replaceChar: []rune(replaceChar),
		root:        nil,
	}
}

func (tree *TrieTree) Match() {

}

func (tree *TrieTree) AddWords(words []string) {
	for _, w := range words {
		tree.AddWord(w)
	}
}

func (tree *TrieTree) AddWord(w string) {
	word := tree.FilterChar(w)
	var node *TrieNode
	for _, r := range []rune(word) {
		node = tree.root.addChild(r)
	}
	node.End = true
	node.Data = w
}

func (tree *TrieTree) FilterChar(w string) string {
	str := strings.ToLower(w)
	str = strings.Replace(" ", "", str, -1)

	regrep := regexp.MustCompile("[^\\u4e00-\\u9fa5a-zA-Z0-9]")
	return regrep.ReplaceAllString(w, "")
}

// addChild 新增子节点
func (tn *TrieNode) addChild(r rune) *TrieNode {
	if tn.ChildMap == nil {
		tn.ChildMap = make(map[rune]*TrieNode)
	}

	if node, ok := tn.ChildMap[r]; ok {
		// 存在就不添加
		return node
	} else {
		tn.ChildMap[r] = &TrieNode{
			ChildMap: nil,
			End:      false,
		}
		return tn.ChildMap[r]
	}
}

// findChild 查找子节点
func (tn *TrieNode) findChild(r rune) *TrieNode {
	if tn.ChildMap == nil {
		return nil
	}

	if node, ok := tn.ChildMap[r]; ok {
		return node
	}
	return nil
}
