package nav

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Page struct {
	URLPath string
	Title   string
	Order   int
	Nav     bool
}

type Entry struct {
	Title    string
	Path     string
	Children []*Entry
}

func Build(pages []Page) []*Entry {
	root := newBuilderNode()

	for i := range pages {
		p := &pages[i]
		if !p.Nav {
			continue
		}
		segments := splitPath(p.URLPath)
		if len(segments) == 0 {
			continue
		}

		cur := root
		for _, s := range segments[:len(segments)-1] {
			cur = cur.child(s)
		}
		leaf := cur.child(segments[len(segments)-1])
		leaf.title = p.Title
		leaf.path = p.URLPath
		leaf.hasPage = true
		leaf.order = p.Order
	}

	return root.entries()
}

func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

type builderNode struct {
	title    string
	path     string
	hasPage  bool
	order    int
	children map[string]*builderNode
	keys     []string
}

func newBuilderNode() *builderNode {
	return &builderNode{children: make(map[string]*builderNode)}
}

func (n *builderNode) child(key string) *builderNode {
	c, ok := n.children[key]
	if !ok {
		c = newBuilderNode()
		n.children[key] = c
		n.keys = append(n.keys, key)
	}
	return c
}

func (n *builderNode) entries() []*Entry {
	if len(n.keys) == 0 {
		return nil
	}

	type ordered struct {
		order int
		entry *Entry
	}

	list := make([]ordered, 0, len(n.keys))
	for _, key := range n.keys {
		c := n.children[key]
		title := c.title
		if title == "" {
			title = titleFromSlug(key)
		}
		list = append(list, ordered{
			order: c.order,
			entry: &Entry{
				Title:    title,
				Path:     c.path,
				Children: c.entries(),
			},
		})
	}

	sort.SliceStable(list, func(i, j int) bool {
		if list[i].order != list[j].order {
			return list[i].order < list[j].order
		}
		return strings.ToLower(list[i].entry.Title) < strings.ToLower(list[j].entry.Title)
	})

	entries := make([]*Entry, len(list))
	for i, o := range list {
		entries[i] = o.entry
	}
	return entries
}

func titleFromSlug(slug string) string {
	words := strings.Split(slug, "-")
	for i, w := range words {
		if w == "" {
			continue
		}
		r, size := utf8.DecodeRuneInString(w)
		words[i] = string(unicode.ToUpper(r)) + w[size:]
	}
	return strings.Join(words, " ")
}
