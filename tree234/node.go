package tree234

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

type Node234[T any] struct {
	parent *Node234[T]
	kids   [4]*Node234[T]
	counts [4]int
	elems  [3]*T
}

func (n *Node234[T]) String() string {
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

func (n *Node234[T]) KidsCount() (c int) {
	if n == nil {
		return
	}
	for c < 4 && n.kids[c] != nil {
		c++
	}
	return
}

func (n *Node234[T]) Count() (c int) {
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

func (n *Node234[T]) Size() (s int) {
	if n == nil {
		return
	}
	for s < 3 && n.elems[s] != nil {
		s++
	}
	return
}

func (n *Node234[T]) ChildIndex() int {
	if n != nil && n.parent != nil {
		for i, kid := range n.parent.kids {
			if n == kid {
				return i
			}
		}
	}
	return -1
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
func (n *Node234[T]) transSubtreeRight(ki int, k, index *int) {
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
func (n *Node234[T]) transSubtreeLeft(ki int, k, index *int) {
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
func (n *Node234[T]) transSubtreeMerge(ki int, k, index *int) {
	var (
		left     = n.kids[ki]
		right    = n.kids[ki+1]
		leftlen  = n.counts[ki]
		rightlen = n.counts[ki+1]
		lsize    = left.Size()
		rsize    = right.Size()
	)

	Log.WithFields(logrus.Fields{
		"op":     "transSubtreeMerge",
		"ki":     ki,
		"parent": n,
		"left":   left,
		"right":  right,
	}).Debug("before")

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
