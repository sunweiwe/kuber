package route

import (
	"fmt"
	"strings"
)

type ElementKind string

const (
	ElementKindNone     ElementKind = ""
	ElementKindConst    ElementKind = "const"
	ElementKindVariable ElementKind = "{}"
	ElementKindStar     ElementKind = "*"
	ElementKindSplit    ElementKind = "/"
)

type Element struct {
	kind  ElementKind
	param string
}

type CompileError struct {
	Pattern  string
	Position int
	Rune     rune
	Message  string
}

func (e CompileError) Error() string {
	return fmt.Sprintf("invalid char [%c] in [%s] at position %d: %s", e.Rune, e.Pattern, e.Position, e.Message)
}

func CompileSection(pattern string) ([]Element, error) {
	elements := []Element{}

	patternLen := len(pattern)
	hasStarSuffix := false
	if pattern[patternLen-1] == '*' {
		pattern = pattern[:patternLen-1]
		hasStarSuffix = true
	}

	pos := 0
	currentKind := ElementKindNone
	for i, rune := range pattern {
		switch {
		case rune == '{' && currentKind != ElementKindVariable:
			if currentKind == ElementKindConst {
				elements = append(elements, Element{kind: ElementKindConst, param: pattern[pos:i]})
			}
			currentKind = ElementKindVariable
			pos = i + 1
		case rune == '}' && currentKind == ElementKindVariable:
			elements = append(elements, Element{kind: ElementKindVariable, param: pattern[pos:i]})
			currentKind = ElementKindNone
			pos = i + 1

		default:
			if currentKind == ElementKindVariable || currentKind == ElementKindConst {
				continue
			}
			if currentKind != ElementKindConst {
				currentKind = ElementKindConst
				pos = i
			}
		}
	}

	switch currentKind {
	case ElementKindConst:
		elements = append(elements, Element{kind: ElementKindConst, param: pattern[pos:]})
	case ElementKindVariable:
		return nil, CompileError{Position: len(pattern), Pattern: pattern, Rune: rune(pattern[len(pattern)-1]), Message: "variable definition not closed"}
	}

	if hasStarSuffix {
		elements = append(elements, Element{kind: ElementKindStar})
	}

	return elements, nil
}

func MatchSection(complied []Element, sections []string) (bool, bool, map[string]string) {
	vars := map[string]string{}
	if len(sections) == 0 {
		return false, false, nil
	}

	section := sections[0]
	pos := 0
	for i, e := range complied {
		switch e.kind {
		case ElementKindConst:
			l := len(e.param)
			if len(section) < pos+l {
				return false, false, nil
			}
			pos += l

		case ElementKindVariable:
			if i == len(complied)-1 {
				vars[e.param] = section[pos:]
				return true, false, vars
			}
			sec := complied[i+1]
			switch sec.kind {
			case ElementKindConst:
				index := strings.Index(section[pos:], sec.param)
				if index == -1 || index == 0 {
					return false, false, nil
				}
				vars[e.param] = section[pos : pos+index]
				pos += index
			case ElementKindVariable:
				continue
			case ElementKindStar:
				vars[e.param] = strings.Join(append([]string{section[pos:]}, sections[1:]...), "")
			}
		case ElementKindStar:
			return true, true, vars
		case ElementKindSplit:
			if section == "/" {
				return true, false, vars
			}
			return false, false, nil
		}

	}

	if section[pos:] != "" {
		return false, false, nil
	}
	return true, false, vars
}
