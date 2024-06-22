/*
 * tree234.c: reasonably generic counted 2-3-4 tree routines.
 *
 * This file is copyright 1999-2001 Simon Tatham.
 *
 * Permission is hereby granted, free of charge, to any person
 * obtaining a copy of this software and associated documentation
 * files (the "Software"), to deal in the Software without
 * restriction, including without limitation the rights to use,
 * copy, modify, merge, publish, distribute, sublicense, and/or
 * sell copies of the Software, and to permit persons to whom the
 * Software is furnished to do so, subject to the following
 * conditions:
 *
 * The above copyright notice and this permission notice shall be
 * included in all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
 * EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
 * OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
 * NONINFRINGEMENT.  IN NO EVENT SHALL SIMON TATHAM BE LIABLE FOR
 * ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF
 * CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
 * CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package main

import "github.com/sirupsen/logrus"

type Node234[T any] struct {
	parent *Node234[T]
	kids   [4]*Node234[T]
	counts [4]int
	elems  [3]*T
}

func (n Node234[T]) size() int {
	if n.elems[2] != nil {
		return 2
	}
	if n.elems[1] != nil {
		return 1
	}
	return 0
}

func countNode234[T any](n *Node234[T]) (res int) {
	if n == nil {
		return
	}
	for _, cnt := range n.counts {
		res += cnt
	}
	for _, elem := range n.elems {
		if elem != nil {
			res++
		}
	}
	return
}

type cmpfn234[T any] func(x, y *T) int

type Tree234[T any] struct {
	root *Node234[T]
	cmp  cmpfn234[T]
}

func NewTree234[T any](cmp cmpfn234[T]) *Tree234[T] {
	return &Tree234[T]{
		root: nil,
		cmp:  cmp,
	}
}

func (t Tree234[T]) Count() int {
	if t.root == nil {
		return 0
	}
	return countNode234(t.root)
}

func (t *Tree234[T]) add234Insert(
	left *Node234[T],
	e *T,
	right *Node234[T],
	n *Node234[T],
	ki int,
) (rootSplit bool) {
	var (
		lcount = countNode234(left)
		rcount = countNode234(right)
	)
	for n != nil {
		if n.elems[1] == nil {
			/* insert a 2-node */
			if ki == 0 {
				/* on left */
				n.kids[2], n.counts[2] = n.kids[1], n.counts[1]
				n.elems[1] = n.elems[0]
				n.kids[1], n.counts[1] = right, rcount
				n.elems[0] = e
				n.kids[0], n.counts[0] = left, lcount
			} else {
				/* on right */
				n.kids[2], n.counts[2] = right, rcount
				n.elems[1] = e
				n.kids[1], n.counts[1] = left, lcount
			}
			for _, kid := range n.kids {
				if kid != nil {
					kid.parent = n
				}
			}
			break
		} else if n.elems[2] == nil {
			/* insert a 3-node */
			if ki == 0 {
				/* on left */
				n.kids[3], n.counts[3] = n.kids[2], n.counts[2]
				n.elems[2] = n.elems[1]
				n.kids[2], n.counts[2] = n.kids[1], n.counts[1]
				n.elems[1] = n.elems[0]
				n.kids[1], n.counts[1] = right, rcount
				n.elems[0] = e
				n.kids[0], n.counts[0] = left, lcount
			} else if ki == 1 {
				/* in middle */
				n.kids[3], n.counts[3] = n.kids[2], n.counts[2]
				n.elems[2] = n.elems[1]
				n.kids[2], n.counts[2] = right, rcount
				n.elems[1] = e
				n.kids[1], n.counts[1] = left, lcount
			} else {
				/* on right */
				n.kids[3], n.counts[3] = right, rcount
				n.elems[2] = e
				n.kids[2], n.counts[2] = left, lcount
			}
			for _, kid := range n.kids {
				if kid != nil {
					kid.parent = n
				}
			}
			break
		} else {
			var m = &Node234[T]{parent: n.parent}
			/*
				insert a 4-node:
				split into 2-node and 3-node
				(by choice 3-node goes first)
			*/
			if ki == 0 {
				m.kids[0], m.counts[0] = left, lcount
				m.elems[0] = e
				m.kids[1], m.counts[1] = right, rcount
				m.elems[1] = n.elems[0]
				m.kids[2], m.counts[2] = n.kids[1], n.counts[1]
				e = n.elems[1]
				n.kids[0], n.counts[0] = n.kids[2], n.counts[2]
				n.elems[0] = n.elems[2]
				n.kids[1], n.counts[1] = n.kids[3], n.counts[3]
			} else if ki == 1 {
				m.kids[0], m.counts[0] = n.kids[0], n.counts[0]
				m.elems[0] = n.elems[0]
				m.kids[1], m.counts[1] = left, lcount
				m.elems[1] = e
				m.kids[2], m.counts[2] = right, rcount
				e = n.elems[1]
				n.kids[0], n.counts[0] = n.kids[2], n.counts[2]
				n.elems[0] = n.elems[2]
				n.kids[3], n.counts[3] = n.kids[1], n.counts[1]
			} else if ki == 2 {
				m.kids[0], m.counts[0] = n.kids[0], n.counts[0]
				m.elems[0] = n.elems[0]
				m.kids[1], m.counts[1] = n.kids[1], n.counts[1]
				m.elems[1] = n.elems[1]
				m.kids[2], m.counts[2] = left, lcount
				n.kids[0], n.counts[0] = right, rcount
				n.elems[0] = n.elems[2]
				n.kids[1], n.counts[1] = n.kids[3], n.counts[3]
			} else { /* ki == 3 */
				m.kids[0], m.counts[0] = n.kids[0], n.counts[0]
				m.elems[0] = n.elems[0]
				m.kids[1], m.counts[1] = n.kids[1], n.counts[1]
				m.elems[1] = n.elems[1]
				m.kids[2], m.counts[2] = n.kids[2], n.counts[2]
				n.kids[0], n.counts[0] = left, lcount
				n.elems[0] = e
				n.kids[1], n.counts[1] = right, rcount
				e = n.elems[2]
			}
			m.kids[3], n.kids[3], n.kids[2] = nil, nil, nil
			m.counts[3], n.counts[3], n.counts[2] = 0, 0, 0
			m.elems[2], n.elems[2], n.elems[1] = nil, nil, nil
			for _, kid := range m.kids {
				if kid != nil {
					kid.parent = m
				}
			}
			for _, kid := range n.kids {
				if kid != nil {
					kid.parent = n
				}
			}
		}
		if n.parent != nil {
			switch n {
			case n.parent.kids[0]:
				ki = 0
			case n.parent.kids[1]:
				ki = 1
			case n.parent.kids[2]:
				ki = 2
			default:
				ki = 3
			}
		}
		n = n.parent
	}

	if n != nil {
		for n.parent != nil {
			var childnum int
			switch n {
			case n.parent.kids[0]:
				childnum = 0
			case n.parent.kids[1]:
				childnum = 1
			case n.parent.kids[2]:
				childnum = 2
			default:
				childnum = 3
			}
			n.parent.counts[childnum] = countNode234(n)
			n = n.parent
		}
		return false
	} else {
		t.root = &Node234[T]{}
		t.root.kids[0], t.root.counts[0] = left, lcount
		t.root.elems[0] = e
		t.root.kids[1], t.root.counts[1] = right, rcount
		for _, kid := range t.root.kids {
			if kid != nil {
				kid.parent = t.root
			}
		}
		return true
	}
}

func (t *Tree234[T]) addInternal(e *T, index int) *T {
	var (
		origE *T = e
	)

	if t.root == nil {
		t.root = &Node234[T]{
			parent: nil,
			kids:   [4]*Node234[T]{},
			counts: [4]int{},
			elems:  [3]*T{e},
		}
		return origE
	}

	var (
		n  *Node234[T] = t.root
		ki int
	)
	for n != nil {
		if index >= 0 {
			if n.kids[0] == nil {
				/* leaf node */
				ki = index
			} else {
				/* internal node */
				if index <= n.counts[0] {
					ki = 0
				} else if index -= n.counts[0] + 1; index <= n.counts[1] {
					ki = 1
				} else if index -= n.counts[1] + 1; index <= n.counts[2] {
					ki = 2
				} else if index -= n.counts[2] + 1; index <= n.counts[3] {
					ki = 3
				} else {
					log.WithFields(logrus.Fields{
						"tree": t, "element": e, "index": index,
					}).Fatalf("index out of range")
				}
			}
		} else {
			for i, el := range n.elems {
				if c := t.cmp(e, el); c < 0 {
					ki = i
					break
				} else if c == 0 {
					return el
				} else if i == len(n.elems)-1 {
					ki = 3
				}
			}
		}

		if n.kids[ki] == nil {
			break
		}
		n = n.kids[ki]
	}
	t.add234Insert(nil, e, nil, n, ki)
	return origE
}

func (t *Tree234[T]) Add(e *T) *T {
	return t.addInternal(e, 1)
}

func (t Tree234[T]) Index(index int) *T {
	if t.root == nil { // tree is empty
		return nil
	}
	if index < 0 || index > countNode234(t.root) { // out of range
		return nil
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
	// we should never get here
	return nil
}

type relation uint8

const (
	Eq relation = iota
	Lt
	Le
	Gt
	Ge
)

func (t *Tree234[T]) FindRelPos(
	e *T,
	relation relation,
) (el *T, index int) {
	if t.root == nil {
		return
	}
	cmp := t.cmp
	var (
		n      = t.root
		idx    = 0
		ecount = -1
		cmpret = 0
		kcount int
	)
	if e == nil { // fake comparison if e == nil
		switch relation {
		case Lt:
			cmpret = 1 // e is always greater
		case Gt:
			cmpret = -1 // e is always smaller
		default:
			log.WithFields(logrus.Fields{
				"e": e, "relation": relation,
			}).Fatal("invalid relation as e == nil")
		}
	}
	for {
		for kcount := 0; kcount < 4; kcount++ {
			if kcount >= 3 || n.elems[kcount] == nil {
				break
			}
			c := Iif(cmpret != 0, cmpret, cmp(e, n.elems[kcount]))
			if c < 0 {
				break
			}
			if n.kids[kcount] != nil {
				idx += n.counts[kcount]
			}
			if c == 0 {
				ecount = kcount
				break
			}
			idx++
		}
		if ecount >= 0 {
			break
		}
		if n.kids[kcount] != nil {
			n = n.kids[kcount]
		} else {
			break
		}
	}

	if ecount >= 0 {
		/*
			Element is found: it is n.elems[ecount] at tree index idx.
			If relation is LE, EQ or GE we are done
		*/
		if relation != Lt && relation != Gt {
			return n.elems[ecount], idx
		}
		/*
			Otherwise we do an index lookup for previous or next element.
		*/
		if relation == Lt {
			idx--
		} else {
			idx++
		}
	} else {
		/*
			We searched through the tree and found a place where we would insert
			this node if we wanted: (empty) subtree n.kids[kcount] and it would
			have index idx.

			But the element isn't there. So if our search relation is EQ, we're
			doomed.
		*/
		if relation == Eq {
			return nil, -1
		}

		/*
			Otherwise, we must do an index lookup for idx-1 (if we're going left
			- LE or LT) or index idx (if we're goind right - GE or GT)
		*/
		if relation == Lt || relation == Le {
			idx--
		}

	}
	ret := t.Index(idx)
	return ret, idx
}

func trans234SubtreeRight[T any](n *Node234[T], ki int) (k, index int) {
	var (
		src  = n.kids[ki]
		dest = n.kids[ki+1]
	)
	dest.kids[3], dest.counts[3] = dest.kids[2], src.counts[2]
	dest.elems[2] = dest.elems[1]
	dest.kids[2], dest.counts[2] = dest.kids[1], src.counts[1]
	dest.elems[1] = dest.elems[0]
	dest.kids[1], dest.counts[1] = dest.kids[0], src.counts[0]

	var i int
	if src.elems[2] != nil {
		i = 2
	} else if src.elems[1] != nil {
		i = 1
	} else {
		i = 0
	}

	dest.elems[0] = n.elems[ki]
	n.elems[ki] = src.elems[i]
	src.elems[i] = nil

	dest.kids[0], dest.counts[0] = src.kids[i+1], src.counts[i+1]
	src.kids[i+1], src.counts[i+1] = nil, 0

	if dest.kids[0] != nil {
		dest.kids[0].parent = dest
	}

	adjust := dest.counts[0] + 1
	n.counts[ki] -= adjust
	n.counts[ki+1] += adjust

	srclen := n.counts[ki]

	// if k != 0 {
	if k == ki && index > srclen {
		index -= srclen + 1
		k++
	} else if k == ki+1 {
		index += adjust
	}
	// }
	return
}

func trans234SubtreeLeft[T any](n *Node234[T], ki int) (k, index int) {
	var (
		src  = n.kids[ki]
		dest = n.kids[ki-1]
	)

	var i int
	if dest.elems[1] != nil {
		i = 2
	} else if dest.elems[0] != nil {
		i = 1
	} else {
		i = 0
	}

	dest.elems[i] = n.elems[ki-1]
	n.elems[ki-1] = src.elems[0]

	dest.kids[i+1], dest.counts[i+1] = src.kids[0], src.counts[0]

	if dest.kids[i+1] != nil {
		dest.kids[i+1].parent = dest
	}

	src.kids[0], src.counts[0] = src.kids[1], src.counts[1]
	src.elems[0] = src.elems[1]
	src.kids[1], src.counts[1] = src.kids[2], src.counts[2]
	src.elems[1] = src.elems[2]
	src.kids[2], src.counts[2] = src.kids[3], src.counts[3]
	src.elems[2] = nil
	src.kids[3], src.counts[3] = nil, 0

	adjust := dest.counts[i+1] + 1

	n.counts[ki] -= adjust
	n.counts[ki-1] += adjust

	// if k != 0 {
	if k == ki {
		index -= adjust
		if index < 0 {
			index += n.counts[ki-1] + 1
			ki--
		}
	}
	// }

	return
}

func trans234SubtreeMerge[T any](n *Node234[T], ki int) (k, index int) {
	var (
		left, leftlen   = n.kids[ki], n.counts[ki]
		right, rightlen = n.kids[ki+1], n.counts[ki+1]
		lsize           = left.size()
		rsize           = right.size()
	)

	if lsize == 2 || rsize == 2 {
		log.Fatal("neither side elements must be large")
	}

	left.elems[lsize] = n.elems[ki]

	for i := 0; i < rsize+1; i++ {
		left.kids[lsize+1+i] = right.kids[i]
		left.counts[lsize+1+i] = right.counts[i]
		if left.kids[lsize+1+i] != nil {
			left.kids[lsize+1+i].parent = left
		}
		if i < rsize {
			left.elems[lsize+1+i] = right.elems[i]
		}
	}

	n.counts[ki] += rightlen + 1

	for i := ki + 1; i < 3; i++ {
		n.kids[i], n.counts[i] = n.kids[i+1], n.counts[i+1]
	}
	for i := ki; i < 2; i++ {
		n.elems[i] = n.elems[i+1]
	}
	n.kids[3], n.counts[3] = nil, 0
	n.elems[2] = nil

	// if k != 0 {
	if k == ki+1 {
		k--
		index += leftlen + 1
	} else if k > ki+1 {
		k--
	}
	// }

	return
}

func (t *Tree234[T]) delpos234Internal(index int) (res *T) {
	var (
		n  = t.root
		ki int
	)
	for {
		if index <= n.counts[0] {
			ki = 0
		} else if index -= n.counts[0] + 1; index <= n.counts[1] {
			ki = 1
		} else if index -= n.counts[1] + 1; index <= n.counts[2] {
			ki = 2
		} else if index -= n.counts[2] + 1; index <= n.counts[3] {
			ki = 3
		} else {
			log.Fatalf("this can't happen")
		}

		if n.kids[0] != nil {
			break /* n is a leaf node; we're here! */
		}

		if index == n.counts[ki] {
			if n.elems[ki] == nil {
				log.Panicf("must be a kid before the element")
			}
			ki++
			index = 0
			m := &Node234[T]{}
			for m = n.kids[ki]; m.kids[0] != nil; m = m.kids[0] {
				continue
			}
			res = n.elems[ki-1]
			n.elems[ki-1] = m.elems[0]
		}

		sub := n.kids[ki]
		if sub.elems[1] == nil {
			if ki > 0 && n.kids[ki-1].elems[1] != nil {
				ki, index = trans234SubtreeRight(n, ki-1)
			} else if ki < 3 && n.kids[ki+1] != nil && n.kids[ki+1].elems[1] != nil {
				ki, index = trans234SubtreeLeft(n, ki+1)
			} else {
				var _ki int
				if ki > 0 {
					_ki = ki - 1
				} else {
					_ki = ki
				}
				ki, index = trans234SubtreeMerge(n, _ki)
				sub = n.kids[ki]
				if n.elems[0] == nil {
					t.root = sub
					sub.parent = nil
					n = nil
				}
			}
		}

		if n != nil {
			n.counts[ki]--
		}
		n = sub
	}

	if n.kids[0] != nil {
		log.WithFields(logrus.Fields{
			"n": n,
		}).Fatal("n must be a leaf node")
	}

	if res == nil {
		res = n.elems[ki]
	}

	var i int
	for i = ki; i < 2 && n.elems[i+1] != nil; i++ {
		n.elems[i] = n.elems[i+1]
	}
	n.elems[i] = nil

	if n.elems[0] == nil {
		if n != t.root {
			log.WithFields(logrus.Fields{
				"n": n, "root": t.root,
			}).Fatal("n must be root")
		}
		t.root = nil
	}
	return
}

func (t *Tree234[T]) Delete(e *T) *T {
	el, index := t.FindRelPos(e, Eq)
	if el == nil {
		return nil
	}
	return t.delpos234Internal(index)
}
