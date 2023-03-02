package trietree

import "testing"

func TestTrieTree_Match(t *testing.T) {
	tree := NewTrieTree("*")
	tree.AddWord("fuck")
	tree.AddWord("傻逼")

	sensitiveWords, fs := tree.Match("你是傻逼吗, fuck`aaad")
	t.Log(sensitiveWords)
	t.Log(fs)
}
