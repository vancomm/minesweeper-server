package tree234

import (
	"math/rand/v2"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	Log.SetLevel(logrus.DebugLevel)
	Log.SetFormatter(&logrus.TextFormatter{DisableSorting: true})
	m.Run()
}

type checkCtx struct {
	treeDepth int
	elemCount int
}

func checkNode[T any](
	t *testing.T, ctx *checkCtx, level int,
	node *Node234[T],
	lowbound *T, highbound *T,
	cmp CompareFunc[T],
) int {
	// t.Helper()

	// [[[[0 1](2) 2 [5](1) 6 [7](1) 8 [E H](2)](9) I [[K](1) M [O](1)](3) P [[R](1) T [U W](2) Z [e g](2)](7)](21) h [[[j m](2) o [p](1)](4) q [[r s](2) w [x](1)](4)](9)]

	/* Count the non-NULL kids. */
	nkids := node.kidsCount()
	/* Ensure no kids beyond the first NULL are non-NULL. */
	for i := nkids; i < 4; i++ {
		require.Nil(t, node.kids[i],
			"node %s: nkids=%d but kids[%d] non-nil",
			node.String(), nkids, i)
		require.Zero(t, node.counts[i],
			"node %s: kids[%d] nil but count[%d]=%d nonzero",
			node.String(), i, i, node.counts[i])
	}

	/* Count the non-NULL elements. */
	nelems := node.size()
	/* Ensure no elements beyond the first NULL are non-NULL. */
	for i := nelems; i < 3; i++ {
		require.Nil(t, node.elems[i],
			"node %s: nelems=%d but elems[%d] non-nil",
			node.String(), nelems, i)
	}

	if nkids == 0 {
		/*
		 * If nkids==0, this is a leaf node; verify that the tree
		 * depth is the same everywhere.
		 */
		if ctx.treeDepth < 0 {
			ctx.treeDepth = level /* we didn't know the depth yet */
		} else {
			require.Equal(t, level, ctx.treeDepth,
				"node %s: leaf at depth %d, previously seen depth %d",
				node.String(), level, ctx.treeDepth)
		}
	} else {
		/*
		 * If nkids != 0, then it should be nelems+1, unless nelems
		 * is 0 in which case nkids should also be 0 (and so we
		 * shouldn't be in this condition at all).
		 */
		expectedKids := iif(nelems == 0, 0, nelems+1)
		require.Equal(t, expectedKids, nkids,
			"node %s: %d elems should mean %d kids but has %d",
			node.String(), nelems, expectedKids, nkids)
	}

	/*
	 * nelems should be at least 1.
	 */
	require.NotZero(t, nelems,
		"node %s: no elems but %d kids",
		node.String(), nkids,
	)

	/*
	 * Add nelems to the running element count of the whole tree.
	 */
	ctx.elemCount += nelems

	/*
	 * Check ordering property: all elements should be strictly >
	 * lowbound, strictly < highbound, and strictly < each other in
	 * sequence. (lowbound and highbound are NULL at edges of tree
	 * - both NULL at root node - and NULL is considered to be <
	 * everything and > everything. IYSWIM.)
	 */
	for i := -1; i < nelems; i++ {
		var lower, higher *T
		if i == -1 {
			lower = lowbound
		} else {
			lower = node.elems[i]
		}
		if i+1 == nelems {
			higher = highbound
		} else {
			higher = node.elems[i+1]
		}
		if lower != nil && higher != nil {
			require.Negative(t, cmp(lower, higher),
				"node %s: kid comparison [%d=%v,%d=%v] failed",
				node.String(), i, lower, i+1, higher)
		}
	}

	/*
	 * Check parent pointers: all non-NULL kids should have a
	 * parent pointer coming back to this node.
	 */
	for i := range nkids {
		require.Equal(t, node, node.kids[i].parent,
			"node %s kid %i: parent is %v not %v",
			node.String(), i, node.kids[i].parent, node)
	}

	/*
	 * Now (finally!) recurse into subtrees.
	 */
	count := nelems

	for i := range nkids {
		var lower, higher *T
		if i == 0 {
			lower = lowbound
		} else {
			lower = node.elems[i-1]
		}
		if i == nelems {
			higher = highbound
		} else {
			higher = node.elems[i]
		}
		subcount := checkNode(t, ctx, level+1, node.kids[i], lower, higher, cmp)
		require.Equal(t, node.counts[i], subcount,
			"node %s kid %d: count says %d, subtree really has %d",
			node.String(), i, node.counts[i], subcount)
		count += subcount
	}

	return count
}

func verifyTree[T any](t *testing.T, tree *Tree234[T], array []*T) {
	// t.Helper()

	ctx := &checkCtx{
		treeDepth: -1, /* depth unknown yet */
		elemCount: 0,  /* no elements seen yet */
	}

	/*
	 * Verify validity of tree properties.
	 */

	if tree.root != nil {
		require.Nil(t, tree.root.parent,
			"root.parent %v should be nil",
			tree.root.parent)
		checkNode(t, ctx, 0, tree.root, nil, nil, tree.cmp)
	}

	t.Logf("tree depth: %d\n", ctx.treeDepth)

	/*
	 * Enumerate the tree and ensure it matches up to the array.
	 */

	i := 0
	for i = 0; tree.Index(i) != nil; i++ {
		require.Less(t, i, len(array),
			"tree contains more than %d elements",
			len(array))
		require.Equal(t, array[i], tree.Index(i),
			"enum at position %d: array says %v, tree says %v",
			i, array[i], tree.Index(i))
	}

	require.Equal(t, ctx.elemCount, i,
		"tree really contains %d elements, enum gave %d",
		ctx.elemCount, i)

	require.GreaterOrEqual(t, i, len(array),
		"enum gave only %d elements, array has %d",
		i, len(array))

	i = tree.Count()
	require.Equal(t, ctx.elemCount, i,
		"tree really contains %d elements, tree.Count gave %d",
		ctx.elemCount, i)
}

func addTestInternal[T any](t *testing.T, tree *Tree234[T], elem *T, index int, realret *T, array *[]*T) {
	// t.Helper()

	i := index
	retval := elem

	*array = insert(*array, i, elem)

	// for j := len(array); j > i; j-- {
	// 	array[j] = array[j-1]
	// }
	// array[i] = elem

	require.Equal(t, realret, retval,
		"add: retval was %v expected %v",
		realret, retval)

	verifyTree(t, tree, *array)
}

func addTest[T any](t *testing.T, tree *Tree234[T], elem *T, array *[]*T) {
	// t.Helper()

	realret := tree.Add(elem)

	i := 0
	for i < len(*array) && tree.cmp(elem, (*array)[i]) > 0 {
		i++
	}

	if i < len(*array) && tree.cmp(elem, (*array)[i]) == 0 {
		require.Equal(t, (*array)[i], realret,
			"add: retval was %v expected %v", realret, (*array)[i])
	} else {
		addTestInternal(t, tree, elem, i, realret, array)
	}
}

func deletePosTest[T any](t *testing.T, tree *Tree234[T], i int, array *[]*T) {
	// t.Helper()

	elem := (*array)[i]

	*array = remove(*array, i) /* delete elem from array */

	ret := tree.Delete(elem)

	require.Equal(t, ret, elem,
		"delete returned %v, expected %v", ret, elem)

	verifyTree(t, tree, *array)
}

func deleteTest[T any](t *testing.T, tree *Tree234[T], elem *T, array *[]*T) {
	// t.Helper()

	i := 0
	for i < len(*array) && tree.cmp(elem, (*array)[i]) > 0 {
		i++
	}
	if i >= len(*array) || tree.cmp(elem, (*array)[i]) != 0 {
		return /* don't do it! */
	}

	deletePosTest(t, tree, i, array)
}

type Element string

func (e *Element) String() string {
	return "'" + string(*e) + "'"
}

func elementCmp(a, b *Element) int {
	aa, bb := *a, *b
	if aa < bb {
		return -1
	}
	if aa > bb {
		return 1
	}
	return 0
}

var elements = [...]Element{
	"0", "2", "3", "I", "K", "d", "H", "J", "Q", "N", "n", "q", "j", "i",
	"7", "G", "F", "D", "b", "x", "g", "B", "e", "v", "V", "T", "f", "E",
	"S", "8", "A", "k", "X", "p", "C", "R", "a", "o", "r", "O", "Z", "u",
	"6", "1", "w", "L", "P", "M", "c", "U", "h", "9", "t", "5", "W", "Y",
	"m", "s", "l", "4",
}

func TestSuite(t *testing.T) {
	var (
		r     = rand.New(rand.NewPCG(1, 2))
		tree  = New(elementCmp)
		in    [len(elements)]bool
		array []*Element
	)

	verifyTree(t, tree, array)

	var (
		rels = map[Relation]string{
			Eq: "EQ", Ge: "GE", Le: "LE", Lt: "LT", Gt: "GT",
		}
		findTest = func() {
			for _, p := range elements {
				for rel, relName := range rels {
					var (
						lo     = 0
						hi     = len(array) - 1
						c, mid int
						ret    *Element
					)
					for lo <= hi {
						mid = (lo + hi) / 2
						c = tree.cmp(&p, array[mid])
						if c < 0 {
							hi = mid - 1
						} else if c > 0 {
							lo = mid + 1
						} else {
							break
						}
					}

					if c == 0 {
						switch rel {
						case Lt:
							if mid > 0 {
								mid -= 1
								ret = array[mid]
							} else {
								ret = nil
							}
						case Gt:
							if mid < len(array)-1 {
								mid += 1
								ret = array[mid]
							} else {
								ret = nil
							}
						default:
							ret = array[mid]
						}
					} else {
						switch rel {
						case Lt, Le:
							mid = hi
							if hi >= 0 {
								ret = array[hi]
							} else {
								ret = nil
							}
						case Gt, Ge:
							mid = lo
							if lo < len(array) {
								ret = array[lo]
							} else {
								ret = nil
							}
						default:
							ret = nil
						}
					}

					realret, index := tree.FindRelPos(&p, rel)

					require.Equal(t, realret, ret,
						"find(\"%s\", %s) gave %s should be %s",
						p, relName, realret, ret)

					if realret != nil {
						require.Equal(t, index, mid,
							"find(\"%s\", %s) gave %d should be %d",
							p, relName, index, mid)

						if rel == Eq {
							realret2 := tree.Index(index)
							require.Equal(t, realret2, realret,
								"find(\"%s\", %s) gave %s(%d) but %d -> %s",
								p, relName, realret, index, index, realret2)
						}
					}
				}
			}

			{
				realret, index := tree.FindRelPos(nil, Gt)
				if len(array) > 0 {
					require.True(t, realret == array[0] && index == 0,
						"find(nil, Gt) gave %s(%d) should be %s(0)",
						realret, index, array[0])
				} else {
					require.Nil(t, realret,
						"find(nil, Gt) gave %s(%d), should be nil",
						realret, index)
				}
			}

			{
				realret, index := tree.FindRelPos(nil, Lt)
				if len(array) > 0 {
					require.True(t, realret == array[len(array)-1] && index == len(array)-1,
						"find(nil, Lt) gave %s(%d) should be %s(0)",
						realret, index, array[len(array)-1])
				} else {
					require.Nil(t, realret,
						"find(nil, Lt) gave %s(%d), should be nil",
						realret, index)
				}
			}
		}
	)

	for i := range 10000 {
		t.Logf("trial: %d\n", i)
		j := r.IntN(len(elements))
		if in[j] {
			t.Logf("deleting %v (%d)\n", elements[j], j)
			deleteTest(t, tree, &elements[j], &array)
			in[j] = false
		} else {
			t.Logf("adding %v (%d)\n", elements[j], j)
			addTest(t, tree, &elements[j], &array)
			in[j] = true
		}
		findTest()
	}
}
