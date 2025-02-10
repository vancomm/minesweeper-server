package tree234

import (
	"fmt"
	"strings"
)

type node234[T any] struct {
	parent *node234[T]
	kids   [4]*node234[T]
	counts [4]int
	elems  [3]*T
}

func (n *node234[T]) kidsCount() (c int) {
	if n == nil {
		return
	}
	for c < 4 && n.kids[c] != nil {
		c++
	}
	return
}

func (n *node234[T]) count() (c int) {
	if n == nil {
		return
	}
	for _, count := range n.counts {
		c += count
	}
	for _, elem := range n.elems {
		if elem != nil {
			c++
		}
	}
	return
}

func (n *node234[T]) size() (s int) {
	if n == nil {
		return
	}
	for s < 3 && n.elems[s] != nil {
		s++
	}
	return
}

func (n *node234[T]) childIndex() int {
	if n != nil && n.parent != nil {
		for i, kid := range n.parent.kids {
			if n == kid {
				return i
			}
		}
	}
	return -1
}

// node234 implements [fmt.Stringer]
func (n *node234[T]) String() string {
	if n == nil {
		return "<nil>"
	}
	var parts []string
	for i := range 4 {
		if n.kids[i] != nil || n.counts[i] > 0 {
			parts = append(parts, fmt.Sprintf("%s(%d)", n.kids[i].String(), n.counts[i]))
		}
		if i < 3 && n.elems[i] != nil {
			if s, ok := any(n.elems[i]).(fmt.Stringer); ok {
				parts = append(parts, s.String())
			} else {
				parts = append(parts, fmt.Sprintf("%v", n.elems[i]))
			}
		}
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, " "))
}
