package tree234

import "github.com/sirupsen/logrus"

type Relation uint8

const (
	Eq Relation = iota
	Lt
	Le
	Gt
	Ge
)

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
