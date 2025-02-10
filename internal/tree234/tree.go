package tree234

import (
	"github.com/sirupsen/logrus"
)

var Log = logrus.New()

type CompareFunc[T any] func(x, y *T) int

type Tree234[T any] struct {
	root *node234[T]
	cmp  CompareFunc[T]
}

func NewTree234[T any](cmp CompareFunc[T]) *Tree234[T] {
	return &Tree234[T]{
		root: nil,
		cmp:  cmp,
	}
}

// Tree234 implements [fmt.Stringer]
func (t Tree234[T]) String() string {
	return t.root.String()
}

func (t Tree234[T]) Count() int {
	return t.root.count()
}

/*
Look up the element at a given numeric index in a 2-3-4 tree.
Returns NULL if the index is out of range.
*/
func (t *Tree234[T]) Index(index int) *T {
	if t.root == nil {
		return nil /* tree is empty */
	}

	if index < 0 || index >= t.root.count() {
		return nil /* out of range */
	}

	n := t.root

	for n != nil {
		if index < n.counts[0] {
			n = n.kids[0]
		} else if index -= n.counts[0] + 1; index < 0 {
			return n.elems[0]
		} else if index < n.counts[1] {
			n = n.kids[1]
		} else if index -= n.counts[1] + 1; index < 0 {
			return n.elems[1]
		} else if index < n.counts[2] {
			n = n.kids[2]
		} else if index -= n.counts[2] + 1; index < 0 {
			return n.elems[2]
		} else {
			n = n.kids[3]
		}
	}

	/* We shouldn't ever get here. I wonder how we did. */
	return nil
}
