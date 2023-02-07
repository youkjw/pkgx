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

func (tree *TrieTree) Match(text string) (sensitiveWords []string, replacedText string) {
	if tree.root == nil {
		return nil, text
	}

	// 过滤特殊字符
	var (
		ftext        = tree.FilterChar(text)
		sensitiveMap = make(map[string]*struct{}) //利用map进行敏感词去重
	)
	stext := []rune(ftext)
	for key, val := range stext {
		trieNode := tree.root.findChild(val)
		if trieNode == nil {
			continue
		}

		// 匹配到首个敏感词
		// 继续匹配后续的敏感词
		for end := key + 1; trieNode != nil; end++ {
			if trieNode.End {
				// 匹配到完整敏感词
				if _, ok := sensitiveMap[trieNode.Data]; !ok {
					sensitiveWords = append(sensitiveWords, trieNode.Data)
				}
				sensitiveMap[trieNode.Data] = nil
				tree.replaceRune(stext, key, end)
			}
			trieNode = trieNode.findChild(stext[end])
		}
	}

	if len(sensitiveWords) > 0 {
		// 有敏感词
		replacedText = string(stext)
	} else {
		// 没有则返回原来的文本
		replacedText = text
	}

	return
}

func (tree *TrieTree) replaceRune(r []rune, start int, end int) {
	l := len(r)
	for i := start; i < end; i++ {
		for k, replaceRune := range tree.replaceChar {
			i = i + k
			end = end + k
			if k == 0 {
				r[i] = replaceRune
			} else {
				// 如果替换的index大于原来的总长度
				// 往最后面添加
				if i > l {
					r = append(r, replaceRune)
				} else {
					// 否则往中间加
					r = append(r[:i], replaceRune)
					r = append(r, r[i:]...)
				}
			}
		}

	}
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
