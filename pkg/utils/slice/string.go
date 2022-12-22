package slice

import (
	"sort"

	"github.com/google/go-cmp/cmp"
)

// ContainStr src contains dest
func ContainStr(src []string, dest string) bool {
	for i := range src {
		if src[i] == dest {
			return true
		}
	}
	return false
}

func StringArrayEqual(s1, s2 []string) bool {
	trans := cmp.Transformer("Sort", func(in []string) []string {
		out := append([]string(nil), in...)
		sort.Strings(out)
		return out
	})

	x := struct{ Strings []string }{s1}
	y := struct{ Strings []string }{s2}
	return cmp.Equal(x, y, trans)
}
