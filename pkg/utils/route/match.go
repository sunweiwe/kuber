package route

import (
	"fmt"
	"sort"
)

type matcher struct {
	root *node
}

type node struct {
	key      []Element
	value    *matchItem
	children []*node
}

func (n *node) indexChild(s *node) int {
	for index, child := range n.children {
		if isSamePattern(child.key, s.key) {
			return index
		}
	}
	return -1
}

func isSamePattern(a, b []Element) bool {
	toString := func(elements []Element) string {
		str := ""
		for _, e := range elements {
			switch e.kind {
			case ElementKindConst:
				str += e.param
			case ElementKindVariable:
				str += "{}"
			case ElementKindStar:
				str += "*"
			case ElementKindSplit:
				str += "/"
			}
		}
		return str
	}
	return toString(a) == toString(b)
}

func sortSectionMatches(sections []*node) {
	sort.Slice(sections, func(i, j int) bool {
		si, sj := sections[i].key, sections[j].key

		switch li, lj := (si)[len(si)-1].kind, (sj)[len(sj)-1].kind; {
		case li == ElementKindStar && lj != ElementKindStar:
			return false
		case li != ElementKindStar && lj == ElementKindStar:
			return true
		}

		ci, cj := 0, 0
		for _, v := range si {
			switch v.kind {
			case ElementKindConst:
				ci += 99
			case ElementKindVariable:
				ci -= 1
			}
		}

		for _, v := range sj {
			switch v.kind {
			case ElementKindConst:
				cj += 99
			case ElementKindVariable:
				cj -= 1
			}
		}

		return ci > cj
	})
}

type matchItem struct {
	pattern string
	value   interface{} // of val not nil. it's the matched
}

func (m *matcher) Register(pattern string, value interface{}) error {
	sections, err := CompilePathPattern(pattern)
	if err != nil {
		return err
	}
	item := &matchItem{pattern: pattern, value: value}

	cur := m.root
	for i, section := range sections {
		child := &node{key: section}
		if index := cur.indexChild(child); index == -1 {
			if i == len(sections)-1 {
				child.value = item
			}
			child.children = append(child.children, child)
			sortSectionMatches(child.children)
		} else {
			child = cur.children[index]
			if i == len(sections)-1 {
				if child.value != nil {
					return fmt.Errorf("pattern %s conflicts with exists %s", pattern, child.value.pattern)
				}
				child.value = item
			}
			cur = child
		}
	}

	return nil
}

func (m *matcher) Match(path string) (bool, interface{}, map[string]string) {
	pathTokens := ParsePathTokens(path)

	vars := map[string]string{}
	match := matchChildren(m.root, pathTokens, vars)
	if match == nil {
		return false, nil, vars
	}
	return true, match.value, vars
}

func matchChildren(cur *node, tokens []string, vars map[string]string) *matchItem {
	if len(tokens) == 0 {
		return nil
	}

	var matched *matchItem
	// TODO
	for _, child := range cur.children {
		if matched, matchTokens, secVars := MatchSection(child.key, tokens); matched {
			if child.value != nil && len(tokens) == 1 || matchTokens {
				mergeMap(secVars, vars)
				return child.value
			}
			ret := matchChildren(child, tokens[1:], secVars)
			if ret != nil {
				mergeMap(secVars, vars)
				return ret
			}
		}
	}
	return matched
}

func mergeMap(src, dst map[string]string) map[string]string {
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
