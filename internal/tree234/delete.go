package tree234

import "github.com/sirupsen/logrus"

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

			var m *node234[T]

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
				n.transSubtreeRight(ki-1, &ki, &index)
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
				n.transSubtreeLeft(ki+1, &ki, &index)
			} else {
				/*
				 * ki is small with only small neighbours. Pick a
				 * neighbour and merge with it.
				 */
				if ki > 0 {
					n.transSubtreeMerge(ki-1, &ki, &index)
				} else {
					n.transSubtreeMerge(ki, &ki, &index)
				}
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
