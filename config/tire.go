package config

import (
    "strings"
)

// TrieNode represents a node in the domain trie.
type TrieNode struct {
    Children map[string]*TrieNode
    IsEnd    bool
}

// Trie holds the root of the domain trie.
type Trie struct {
    Root *TrieNode
}

// NewTrie initializes and returns an empty Trie.
func NewTrie() *Trie {
    return &Trie{Root: &TrieNode{Children: make(map[string]*TrieNode)}}
}

// Insert adds a domain into the trie. Domains are split by '.', and labels are inserted in reverse order.
func (t *Trie) Insert(domain string) {
    labels := strings.Split(strings.TrimSuffix(domain, "."), ".")
    node := t.Root
    for i := len(labels) - 1; i >= 0; i-- {
        lbl := labels[i]
        if _, ok := node.Children[lbl]; !ok {
            node.Children[lbl] = &TrieNode{Children: make(map[string]*TrieNode)}
        }
        node = node.Children[lbl]
    }
    node.IsEnd = true
}

func (t *Trie) MatchLongest(domain string) (string, bool) {
    labels := strings.Split(strings.TrimSuffix(domain, "."), ".")
    node := t.Root
    var matchedLabels []string
    var found string
    // Traverse from TLD to subdomain
    for i := len(labels) - 1; i >= 0; i-- {
        lbl := labels[i]
        child, ok := node.Children[lbl]
        if !ok {
            break
        }
        matchedLabels = append([]string{lbl}, matchedLabels...)
        node = child
        if node.IsEnd {
            found = strings.Join(matchedLabels, ".")
        }
    }
    if found == "" {
        return "", false
    }
    return found, true
}

// Print prints the entire trie structure, indenting by level.
func (t *Trie) Print() {
    t.printNode(t.Root, 0)
}

func (t *Trie) printNode(node *TrieNode, level int) {
    for label, child := range node.Children {
        // indent for hierarchy
        indent := strings.Repeat("  ", level)
        if child.IsEnd {
            // mark end of entry
            println(indent + label + " (end)")
        } else {
            println(indent + label)
        }
        t.printNode(child, level+1)
    }
}
