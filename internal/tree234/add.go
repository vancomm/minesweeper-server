package tree234

import "github.com/sirupsen/logrus"

/*
Propagate a node overflow up a tree until it stops. Returns 0 or
1, depending on whether the root had to be split or not.
*/
func (t *Tree234[T]) addInsert(
	left *node234[T],
	e *T,
	right *node234[T],
	n *node234[T],
	ki int,
) (rootSplit bool) {
	log := Log.WithFields(logrus.Fields{
		"op": "addInsert", "left": left, "element": e, "right": right,
		"index": ki, "parent": n,
	})

	/*
		We need to insert the new left/element/right set in n at child point ki.
	*/
	var (
		lcount = left.count()
		rcount = right.count()
	)

	log.Debugf("inserting set")

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
			var m = &node234[T]{parent: n.parent}
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
			lcount = left.count()
			right = n
			rcount = right.count()
		}
		if n.parent != nil {
			ki = n.childIndex()
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
			n.parent.counts[n.childIndex()] = n.count()
			n = n.parent
		}
		return false /* root unchanged */
	} else {
		t.root = &node234[T]{
			kids:   [4]*node234[T]{left, right, nil, nil},
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
		t.root = &node234[T]{
			elems:  [3]*T{e, nil, nil},
			kids:   [4]*node234[T]{nil, nil, nil, nil},
			counts: [4]int{0, 0, 0, 0},
			parent: nil,
		}

		addLog.WithFields(logrus.Fields{
			"root": t.root,
		}).Debug("created new root; done")

		return originalElement
	}

	var (
		n  *node234[T] = t.root
		ki int         = 0
	)

	// do { ... } while (n)
	for ok := true; ok; ok = n != nil {
		if index >= 0 {
			if n.kids[0] == nil {
				/*
				 * Leaf node. We want to insert at kid point
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
