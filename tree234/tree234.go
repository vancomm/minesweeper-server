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

package tree234

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

var Log = logrus.New()

type Relation uint8

const (
	Eq Relation = iota
	Lt
	Le
	Gt
	Ge
)

type Node234[T any] struct {
	parent *Node234[T]
	kids   [4]*Node234[T]
	counts [4]int
	elems  [3]*T
}

func (n Node234[T]) String() string {
	var parts []string
	for i := range 4 {
		if n.kids[i] != nil || n.counts[i] > 0 {
			var kidString string
			if n.kids[i] != nil {
				kidString = n.kids[i].String()
			} else {
				kidString = "<nil>"
			}
			parts = append(parts, fmt.Sprintf("%s(%d)", kidString, n.counts[i]))
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

func (n Node234[T]) kidsCount() int {
	nkids := 0
	for nkids < 4 && n.kids[nkids] != nil {
		nkids++
	}
	return nkids
}

func (n Node234[T]) size() int {
	nelems := 0
	for nelems < 3 && n.elems[nelems] != nil {
		nelems++
	}
	return nelems
}

func (n *Node234[T]) childNum() int {
	if n.parent != nil {
		for i, kid := range n.parent.kids {
			if n == kid {
				return i
			}
		}
	}
	return -1
}

func countNode[T any](n *Node234[T]) int {
	count := 0
	if n == nil {
		return count
	}
	for _, cnt := range n.counts {
		count += cnt
	}
	for _, elem := range n.elems {
		if elem != nil {
			count++
		}
	}
	return count
}

type CompareFunc[T any] func(x, y *T) int

type Tree234[T any] struct {
	root *Node234[T]
	cmp  CompareFunc[T]
}

func (t Tree234[T]) String() string {
	if t.root != nil {
		return t.root.String()
	} else {
		return "<nil>"
	}
}

func New[T any](cmp CompareFunc[T]) *Tree234[T] {
	return &Tree234[T]{
		root: nil,
		cmp:  cmp,
	}
}

func (t Tree234[T]) Count() int {
	if t.root == nil {
		return 0
	}
	return countNode(t.root)
}

/*
Propagate a node overflow up a tree until it stops. Returns 0 or
1, depending on whether the root had to be split or not.
*/
func (t *Tree234[T]) addInsert(
	left *Node234[T],
	e *T,
	right *Node234[T],
	n *Node234[T],
	ki int,
) (rootSplit bool) {
	addInsLog := Log.WithFields(logrus.Fields{
		"op": "addInsert", "left": left, "element": e, "right": right,
		"index": ki, "parent": n,
	})

	/*
		We need to insert the new left/element/right set in n at child position ki.
	*/
	var (
		lcount = countNode(left)
		rcount = countNode(right)
	)

	addInsLog.Debugf("inserting set")

	for n != nil {
		if n.elems[1] == nil {
			/*
			 * Insert in a 2-node; simple.
			 */
			if ki == 0 {
				/* on left */
				n.kids[2] = n.kids[1]
				n.counts[2] = n.counts[1]
				n.elems[1] = n.elems[0]
				n.kids[1] = right
				n.counts[1] = rcount
				n.elems[0] = e
				n.kids[0] = left
				n.counts[0] = lcount
			} else { /* ki == 1 */
				/* on right */
				n.kids[2] = right
				n.counts[2] = rcount
				n.elems[1] = e
				n.kids[1] = left
				n.counts[1] = lcount
			}
			for i := range 3 {
				if n.kids[i] != nil {
					n.kids[i].parent = n
				}
			}
			break
		} else if n.elems[2] == nil {
			/*
			 * Insert in a 3-node; simple.
			 */
			if ki == 0 {
				/* on left */
				n.kids[3] = n.kids[2]
				n.counts[3] = n.counts[2]
				n.elems[2] = n.elems[1]
				n.kids[2] = n.kids[1]
				n.counts[2] = n.counts[1]
				n.elems[1] = n.elems[0]
				n.kids[1] = right
				n.counts[1] = rcount
				n.elems[0] = e
				n.kids[0] = left
				n.counts[0] = lcount
			} else if ki == 1 {
				/* in middle */
				n.kids[3] = n.kids[2]
				n.counts[3] = n.counts[2]
				n.elems[2] = n.elems[1]
				n.kids[2] = right
				n.counts[2] = rcount
				n.elems[1] = e
				n.kids[1] = left
				n.counts[1] = lcount
			} else { /* ki == 2 */
				/* on right */
				n.kids[3] = right
				n.counts[3] = rcount
				n.elems[2] = e
				n.kids[2] = left
				n.counts[2] = lcount
			}
			for i := range 4 {
				if n.kids[i] != nil {
					n.kids[i].parent = n
				}
			}
			break
		} else {
			var m = &Node234[T]{parent: n.parent}
			/*
			 * Insert in a 4-node; split into a 2-node and a
			 * 3-node, and move focus up a level.
			 *
			 * I don't think it matters which way round we put the
			 * 2 and the 3. For simplicity, we'll put the 3 first
			 * always.
			 */
			if ki == 0 {
				m.kids[0] = left
				m.counts[0] = lcount
				m.elems[0] = e
				m.kids[1] = right
				m.counts[1] = rcount
				m.elems[1] = n.elems[0]
				m.kids[2] = n.kids[1]
				m.counts[2] = n.counts[1]
				e = n.elems[1]
				n.kids[0] = n.kids[2]
				n.counts[0] = n.counts[2]
				n.elems[0] = n.elems[2]
				n.kids[1] = n.kids[3]
				n.counts[1] = n.counts[3]
			} else if ki == 1 {
				m.kids[0] = n.kids[0]
				m.counts[0] = n.counts[0]
				m.elems[0] = n.elems[0]
				m.kids[1] = left
				m.counts[1] = lcount
				m.elems[1] = e
				m.kids[2] = right
				m.counts[2] = rcount
				e = n.elems[1]
				n.kids[0] = n.kids[2]
				n.counts[0] = n.counts[2]
				n.elems[0] = n.elems[2]
				n.kids[1] = n.kids[3]
				n.counts[1] = n.counts[3]
			} else if ki == 2 {
				m.kids[0] = n.kids[0]
				m.counts[0] = n.counts[0]
				m.elems[0] = n.elems[0]
				m.kids[1] = n.kids[1]
				m.counts[1] = n.counts[1]
				m.elems[1] = n.elems[1]
				m.kids[2] = left
				m.counts[2] = lcount
				/* e = e; */
				n.kids[0] = right
				n.counts[0] = rcount
				n.elems[0] = n.elems[2]
				n.kids[1] = n.kids[3]
				n.counts[1] = n.counts[3]
			} else { /* ki == 3 */
				m.kids[0] = n.kids[0]
				m.counts[0] = n.counts[0]
				m.elems[0] = n.elems[0]
				m.kids[1] = n.kids[1]
				m.counts[1] = n.counts[1]
				m.elems[1] = n.elems[1]
				m.kids[2] = n.kids[2]
				m.counts[2] = n.counts[2]
				n.kids[0] = left
				n.counts[0] = lcount
				n.elems[0] = e
				n.kids[1] = right
				n.counts[1] = rcount
				e = n.elems[2]
			}
			m.kids[3], n.kids[3], n.kids[2] = nil, nil, nil
			m.counts[3], n.counts[3], n.counts[2] = 0, 0, 0
			m.elems[2], n.elems[2], n.elems[1] = nil, nil, nil
			for i := range 3 {
				if m.kids[i] != nil {
					m.kids[i].parent = m
				}
			}
			for i := range 2 {
				if n.kids[i] != nil {
					n.kids[i].parent = n
				}
			}
			left = m
			lcount = countNode(left)
			right = n
			rcount = countNode(right)
		}
		if n.parent != nil {
			ki = n.childNum()
		}
		n = n.parent
	}

	/*
	 * If we've come out of here by `break', n will still be
	 * non-NULL and all we need to do is go back up the tree
	 * updating counts. If we've come here because n is NULL, we
	 * need to create a new root for the tree because the old one
	 * has just split into two.
	 */
	if n != nil {
		for n.parent != nil {
			n.parent.counts[n.childNum()] = countNode(n)
			n = n.parent
		}
		return false /* root unchanged */
	} else {
		t.root = &Node234[T]{
			kids:   [4]*Node234[T]{left, right, nil, nil},
			counts: [4]int{lcount, rcount, 0, 0},
			elems:  [3]*T{e, nil, nil},
			parent: nil,
		}
		if left != nil {
			left.parent = t.root
		}
		if right != nil {
			right.parent = t.root
		}
		return true /* root moved */
	}
}

/*
Add an element e to a 2-3-4 tree t. Returns e on success, or if an existing
element compares equal, returns that.
*/
func (t *Tree234[T]) addInternal(e *T, index int) *T {
	var (
		addLog = Log.WithFields(logrus.Fields{
			"op": "add", "element": e,
		})
		originalElement *T = e
	)

	addLog.WithFields(logrus.Fields{
		"root": t.root,
	}).Debug("adding element to tree")

	if t.root == nil {
		t.root = &Node234[T]{
			elems:  [3]*T{e, nil, nil},
			kids:   [4]*Node234[T]{nil, nil, nil, nil},
			counts: [4]int{0, 0, 0, 0},
			parent: nil,
		}

		addLog.WithFields(logrus.Fields{
			"root": t.root,
		}).Debug("created new root; done")

		return originalElement
	}

	var (
		n  *Node234[T] = t.root
		ki int         = 0
	)

	// do { ... } while (n)
	for ok := true; ok; ok = n != nil {
		if index >= 0 {
			if n.kids[0] == nil {
				/*
				 * Leaf node. We want to insert at kid position
				 * equal to the index:
				 *
				 *   0 A 1 B 2 C 3
				 */
				ki = index
			} else {
				/*
				 * Internal node. We always descend through it (add
				 * always starts at the bottom, never in the
				 * middle).
				 */
				if index <= n.counts[0] {
					ki = 0
				} else if index -= n.counts[0] + 1; index <= n.counts[1] {
					ki = 1
				} else if index -= n.counts[1] + 1; index <= n.counts[2] {
					ki = 2
				} else if index -= n.counts[2] + 1; index <= n.counts[3] {
					ki = 3
				} else {
					Log.WithFields(logrus.Fields{
						"tree": t, "element": e, "index": index,
					}).Fatalf("index out of range")
				}
			}
		} else {
			if c := t.cmp(e, n.elems[0]); c < 0 {
				ki = 0
			} else if c == 0 {
				return n.elems[0] /* already exists */
			} else if n.elems[1] == nil {
				ki = 1
			} else if c = t.cmp(e, n.elems[1]); c < 0 {
				ki = 1
			} else if c == 0 {
				return n.elems[1] /* already exists */
			} else if n.elems[2] == nil {
				ki = 2
			} else if c = t.cmp(e, n.elems[2]); c < 0 {
				ki = 2
			} else if c == 0 {
				return n.elems[2] /* already exists */
			} else {
				ki = 3
			}
		}

		addLog.WithFields(logrus.Fields{
			"ki": ki, "k": n.kids[ki],
		}).Debug("moving to child")

		if n.kids[ki] == nil {
			break
		}
		n = n.kids[ki]
	}

	t.addInsert(nil, e, nil, n, ki)

	return originalElement
}

func (t *Tree234[T]) Add(e *T) *T {
	return t.addInternal(e, -1)
}

/*
Look up the element at a given numeric index in a 2-3-4 tree.
Returns NULL if the index is out of range.
*/
func (t Tree234[T]) Index(index int) *T {
	if t.root == nil {
		return nil /* tree is empty */
	}

	if index < 0 || index >= countNode(t.root) {
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

/*
Find an element e in a sorted 2-3-4 tree t. Returns NULL if not
found. e is always passed as the first argument to cmp[, so cmp
can be an asymmetric function if desired. cmp can also be passed
as NULL, in which case the compare function from the tree proper
will be used].
*/
func (t *Tree234[T]) FindRelPos(e *T, relation Relation) (el *T, index int) {
	if t.root == nil {
		return
	}

	var (
		n = t.root

		cmpret = 0 /* Prepare a fake `cmp' result if e is NULL. */
	)
	if e == nil {
		switch relation {
		case Lt:
			cmpret = +1 /* e is a max: always greater */
		case Gt:
			cmpret = -1 /* e is a min: always smaller */
		default:
			Log.WithFields(logrus.Fields{
				"e": e, "relation": relation,
			}).Fatal("invalid relation as e == nil")
		}
	}
	var (
		idx    = 0
		ecount = -1
		kcount int
	)
	for {
		for kcount = 0; kcount < 4; kcount++ {
			if kcount >= 3 || n.elems[kcount] == nil {
				break
			}
			var c int
			if cmpret != 0 {
				c = cmpret
			} else {
				c = t.cmp(e, n.elems[kcount])
			}
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
		 * We have found the element we're looking for. It's
		 * n->elems[ecount], at tree index idx. If our search
		 * relation is EQ, LE or GE we can now go home.
		 */
		if relation != Lt && relation != Gt {
			return n.elems[ecount], idx
		}

		/*
		 * Otherwise, we'll do an indexed lookup for the previous
		 * or next element. (It would be perfectly possible to
		 * implement these search types in a non-counted tree by
		 * going back up from where we are, but far more fiddly.)
		 */
		if relation == Lt {
			idx--
		} else {
			idx++
		}
	} else {
		/*
		 * We've found our way to the bottom of the tree and we
		 * know where we would insert this node if we wanted to:
		 * we'd put it in in place of the (empty) subtree
		 * n->kids[kcount], and it would have index idx
		 *
		 * But the actual element isn't there. So if our search
		 * relation is EQ, we're doomed.
		 */
		if relation == Eq {
			return nil, -1
		}

		/*
		 * Otherwise, we must do an index lookup for index idx-1
		 * (if we're going left - LE or LT) or index idx (if we're
		 * going right - GE or GT).
		 */
		if relation == Lt || relation == Le {
			idx--
		}
	}

	/*
	 * We know the index of the element we want; just call index234
	 * to do the rest. This will return NULL if the index is out of
	 * bounds, which is exactly what we want.
	 */
	ret := t.Index(idx)
	return ret, idx
}

/*
 * Tree transformation used in delete and split: move a subtree
 * right, from child ki of a node to the next child. Update k and
 * index so that they still point to the same place in the
 * transformed tree. Assumes the destination child is not full, and
 * that the source child does have a subtree to spare. Can cope if
 * the destination child is undersized.
 *
 *                . C .                     . B .
 *               /     \     ->            /     \
 * [more] a A b B c   d D e      [more] a A b   c C d D e
 *
 *                 . C .                     . B .
 *                /     \    ->             /     \
 *  [more] a A b B c     d        [more] a A b   c C d
 */
func transSubtreeRight[T any](n *Node234[T], ki int, k, index *int) {
	var (
		src  = n.kids[ki]
		dest = n.kids[ki+1]
		log  = Log.WithFields(logrus.Fields{
			"op": "transSubtreeRight",
			"ki": ki,
		})
	)

	log.WithFields(logrus.Fields{
		"parent": n.String(),
		"src":    src,
		"dest":   dest,
	}).Debug("before")

	/*
	 * Move over the rest of the destination node to make space.
	 */
	dest.kids[3] = dest.kids[2]
	dest.kids[2] = dest.kids[1]
	dest.kids[1] = dest.kids[0]
	dest.counts[3] = dest.counts[2]
	dest.counts[2] = dest.counts[1]
	dest.counts[1] = dest.counts[0]
	dest.elems[2] = dest.elems[1]
	dest.elems[1] = dest.elems[0]

	log.WithFields(logrus.Fields{
		"parent": n.String(),
		"src":    src,
		"dest":   dest,
	}).Debug("make space")

	/* which element to move over */
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

	dest.kids[0] = src.kids[i+1]
	dest.counts[0] = src.counts[i+1]
	src.kids[i+1] = nil
	src.counts[i+1] = 0

	if dest.kids[0] != nil {
		dest.kids[0].parent = dest
	}

	log.WithFields(logrus.Fields{
		"parent": n.String(),
		"src":    src,
		"dest":   dest,
	}).Debug("before adjust")

	adjust := dest.counts[0] + 1

	n.counts[ki] -= adjust
	n.counts[ki+1] += adjust

	log.WithFields(logrus.Fields{
		"parent": n.String(),
		"src":    src,
		"dest":   dest,
	}).Debug("after adjust")

	srclen := n.counts[ki]

	if k != nil {
		if (*k) == ki && (*index) > srclen {
			(*index) -= srclen + 1
			(*k)++
		} else if (*k) == ki+1 {
			(*index) += adjust
		}
	}

	log.WithFields(logrus.Fields{
		"parent": n.String(),
		"src":    src,
		"dest":   dest,
	}).Debug("after")
}

/*
 * Tree transformation used in delete and split: move a subtree
 * left, from child ki of a node to the previous child. Update k
 * and index so that they still point to the same place in the
 * transformed tree. Assumes the destination child is not full, and
 * that the source child does have a subtree to spare. Can cope if
 * the destination child is undersized.
 *
 *      . B .                             . C .
 *     /     \                ->         /     \
 *  a A b   c C d D e [more]      a A b B c   d D e [more]
 *
 *     . A .                             . B .
 *    /     \                 ->        /     \
 *   a   b B c C d [more]            a A b   c C d [more]
 */
func transSubtreeLeft[T any](n *Node234[T], ki int, k, index *int) {
	var (
		src  = n.kids[ki]
		dest = n.kids[ki-1]
	)

	/* where in dest to put it */
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

	dest.kids[i+1] = src.kids[0]
	dest.counts[i+1] = src.counts[0]

	if dest.kids[i+1] != nil {
		dest.kids[i+1].parent = dest
	}

	/*
	 * Move over the rest of the source node.
	 */
	src.kids[0] = src.kids[1]
	src.kids[1] = src.kids[2]
	src.kids[2] = src.kids[3]
	src.kids[3] = nil
	src.counts[0] = src.counts[1]
	src.counts[1] = src.counts[2]
	src.counts[2] = src.counts[3]
	src.counts[3] = 0
	src.elems[0] = src.elems[1]
	src.elems[1] = src.elems[2]
	src.elems[2] = nil

	adjust := dest.counts[i+1] + 1

	n.counts[ki] -= adjust
	n.counts[ki-1] += adjust

	if k != nil {
		if (*k) == ki {
			(*index) -= adjust
			if (*index) < 0 {
				(*index) += n.counts[ki-1] + 1
				(*k)--
			}
		}
	}
}

/*
 * Tree transformation used in delete and split: merge child nodes
 * ki and ki+1 of a node. Update k and index so that they still
 * point to the same place in the transformed tree. Assumes both
 * children _are_ sufficiently small.
 *
 *      . B .                .
 *     /     \     ->        |
 *  a A b   c C d      a A b B c C d
 *
 * This routine can also cope with either child being undersized:
 *
 *     . A .                 .
 *    /     \      ->        |
 *   a     b B c         a A b B c
 *
 *    . A .                  .
 *   /     \       ->        |
 *  a   b B c C d      a A b B c C d
 */
func transSubtreeMerge[T any](n *Node234[T], ki int, k, index *int) {
	var (
		left     = n.kids[ki]
		right    = n.kids[ki+1]
		leftlen  = n.counts[ki]
		rightlen = n.counts[ki+1]
		lsize    = left.size()
		rsize    = right.size()
	)

	if lsize == 2 || rsize == 2 {
		Log.Fatal("neither side elements must be large")
	}

	left.elems[lsize] = n.elems[ki]

	for i := range rsize + 1 {
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

	/*
	 * Move the rest of n up by one.
	 */
	for i := ki + 1; i < 3; i++ {
		n.kids[i] = n.kids[i+1]
		n.counts[i] = n.counts[i+1]
	}
	for i := ki; i < 2; i++ {
		n.elems[i] = n.elems[i+1]
	}
	n.kids[3] = nil
	n.counts[3] = 0
	n.elems[2] = nil

	if k != nil {
		if (*k) == ki+1 {
			(*k)--
			(*index) += leftlen + 1
		} else if (*k) > ki+1 {
			(*k)--
		}
	}
}

/*
Delete an element e in a 2-3-4 tree.[ Does not free the element,
merely removes all links to it from the tree nodes.]
*/
func (t *Tree234[T]) deletePosInternal(index int) *T {
	var (
		n   = t.root /* by assumption this is non-NULL */
		res *T
		ki  int
	)

	Log.WithFields(logrus.Fields{
		"index": index, "root": t.root,
	}).Debug("deleting item from tree")

	for {
		log := Log.WithFields(logrus.Fields{
			"node": n, "index": index,
		})

		log.Debug("inspecting node")

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
				"tree": t,
			}).Fatalf("this can't happen") /* can't happen */
		}

		if n.kids[0] == nil {
			break /* n is a leaf node; we're here! */
		}

		/*
		 * Check to see if we've found our target element. If so,
		 * we must choose a new target (we'll use the old target's
		 * successor, which will be in a leaf), move it into the
		 * place of the old one, continue down to the leaf and
		 * delete the old copy of the new target.
		 */
		if index == n.counts[ki] {
			if n.elems[ki] == nil { /* must be a kid _before_ an element */
				log.WithFields(logrus.Fields{
					"index": index, "ki": ki,
				}).Fatalf("there must be a kid before the element")
			}

			var m *Node234[T]

			ki++
			index = 0
			for m = n.kids[ki]; m.kids[0] != nil; m = m.kids[0] {
				continue
			}
			res = n.elems[ki-1]
			n.elems[ki-1] = m.elems[0]
		}

		/*
		 * Recurse down to subtree ki. If it has only one element,
		 * we have to do some transformation to start with.
		 */
		sub := n.kids[ki]
		if sub.elems[1] == nil {
			log := log.WithFields(logrus.Fields{
				"node": n,
			})
			if ki > 0 && n.kids[ki-1].elems[1] != nil {
				/*
				 * Child ki has only one element, but child
				 * ki-1 has two or more. So we need to move a
				 * subtree from ki-1 to ki.
				 */
				log.WithFields(logrus.Fields{
					"ki": ki - 1, "k": ki,
				}).Debug("calling transSubtreeRight")
				transSubtreeRight(n, ki-1, &ki, &index)
			} else if ki < 3 &&
				n.kids[ki+1] != nil &&
				n.kids[ki+1].elems[1] != nil {
				/*
				 * Child ki has only one element, but ki+1 has
				 * two or more. Move a subtree from ki+1 to ki.
				 */
				log.WithFields(logrus.Fields{
					"ki": ki + 1, "k": ki,
				}).Debug("calling transSubtreeLeft")
				transSubtreeLeft(n, ki+1, &ki, &index)
			} else {
				/*
				 * ki is small with only small neighbours. Pick a
				 * neighbour and merge with it.
				 */
				_ki := iif(ki > 0, ki-1, ki)
				log.WithFields(logrus.Fields{
					"ki": _ki, "k": ki,
				}).Debug("calling transSubtreeMerge")
				transSubtreeMerge(n, _ki, &ki, &index)
				sub = n.kids[ki]

				if n.elems[0] == nil {
					/*
					 * The root is empty and needs to be
					 * removed.
					 */
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

	/*
	 * Now n is a leaf node, and ki marks the element number we
	 * want to delete. We've already arranged for the leaf to be
	 * bigger than minimum size, so let's just go to it.
	 */
	if n.kids[0] != nil {
		Log.WithFields(logrus.Fields{
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

	/*
	 * It's just possible that we have reduced the leaf to zero
	 * size. This can only happen if it was the root - so destroy
	 * it and make the tree empty.
	 */
	if n.elems[0] == nil {
		if n != t.root {
			Log.WithFields(logrus.Fields{
				"n": n, "root": t.root,
			}).Fatal("n must be root")
		}
		t.root = nil
	}

	return res /* finished! */
}

func (t *Tree234[T]) DeletePos(index int) *T {
	if index < 0 || index >= t.Count() {
		return nil
	}
	return t.deletePosInternal(index)
}

func (t *Tree234[T]) Delete(e *T) *T {
	el, index := t.FindRelPos(e, Eq)
	if el == nil {
		return nil /* it wasn't in there anyway */
	}
	return t.deletePosInternal(index) /* it's there; delete it. */
}
