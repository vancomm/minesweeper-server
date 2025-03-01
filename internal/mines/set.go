package mines

import (
	"fmt"

	"github.com/vancomm/minesweeper-server/internal/tree234"
)

/*
We use a tree234 to store a large number of small localised
sets, each with a mine count. We also keep some of those sets
linked together into a to-do list.
*/
type set struct {
	x, y       int
	mask       uint16
	mines      int
	todo       bool
	next, prev *set
}

func (s set) String() string {
	return fmt.Sprintf("%d.%d.%d", s.y, s.x, s.mask)
}

func setcmp(a, b *set) int {
	if a.y < b.y {
		return -1
	}
	if a.y > b.y {
		return 1
	}
	if a.x < b.x {
		return -1
	}
	if a.x > b.x {
		return 1
	}
	if a.mask < b.mask {
		return -1
	}
	if a.mask > b.mask {
		return 1
	}
	return 0
}

type setstore struct {
	sets                 *tree234.Tree234[set]
	todo_head, todo_tail *set
}

func newSetStore() *setstore {
	return &setstore{
		sets:      tree234.NewTree234(setcmp),
		todo_head: nil, todo_tail: nil,
	}
}

func (ss *setstore) addTodo(s *set) {
	if s.todo {
		return /* already on it */
	}
	s.prev = ss.todo_tail
	if s.prev != nil {
		s.prev.next = s
	} else {
		ss.todo_head = s
	}
	ss.todo_tail = s
	s.next = nil
	s.todo = true
}

func (ss *setstore) add(x, y int, mask uint16, mines int) error {
	if mask == 0 { // assert mask != 0
		return AssertionError{"mask cannot be 0"}
	}

	/*
	 * Normalise so that x and y are genuinely the bounding
	 * rectangle.
	 */
	for mask&(1|8|64) == 0 {
		mask >>= 1
		x++
	}
	for mask&(1|2|4) == 0 {
		mask >>= 3
		y++
	}

	/*
	 * Create a set structure and add it to the tree.
	 */
	s := &set{
		x:     x,
		y:     y,
		mask:  mask,
		mines: mines,
		todo:  false,
	}

	if ss.sets.Add(s) != s {
		/*
		 * This set already existed! Free it and return.
		 */
		return nil
	}

	/*
	 * We've added a new set to the tree, so put it on the todo
	 * list.
	 */
	ss.addTodo(s)
	return nil
}

/*
 * Remove s from the todo list.
 */
func (ss *setstore) remove(s *set) {
	var (
		next = s.next
		prev = s.prev
	)

	if prev != nil {
		prev.next = next
	} else if s == ss.todo_head {
		ss.todo_head = next
	}

	if next != nil {
		next.prev = prev
	} else if s == ss.todo_tail {
		ss.todo_tail = prev
	}

	s.todo = false

	/*
	 * Remove s from the tree.
	 */
	ss.sets.Delete(s)
}

/*
Return a dynamically allocated list of all the sets which
overlap a provided input set.
*/
func (ss *setstore) overlap(x, y int, mask uint16) (ret []*set) {
	for xx := x - 3; xx < x+3; xx++ {
		for yy := y - 3; yy < y+3; yy++ {
			/*
			 * Find the first set with these top left coordinates.
			 */
			stmp := set{
				x:    xx,
				y:    yy,
				mask: 0,
			}
			if el, p := ss.sets.FindRelPos(&stmp, tree234.Ge); el != nil {
				for s := ss.sets.Index(p); s != nil &&
					s.x == xx && s.y == yy; s = ss.sets.Index(p) {
					/*
					 * This set potentially overlaps the input one.
					 * Compute the intersection to see if they
					 * really overlap, and add it to the list if
					 * so.
					 */
					if setMunge(x, y, mask, s.x, s.y, s.mask, false) != 0 {
						/*
						 * There's an overlap.
						 */
						ret = append(ret, s)
					}
					p++
				}
			}
		}
	}
	return
}

/*
Get an element from the head of the set todo list.
*/
func (ss *setstore) todo() *set {
	if ss.todo_head != nil {
		ret := ss.todo_head
		ss.todo_head = ret.next
		if ss.todo_head != nil {
			ss.todo_head.prev = nil
		} else {
			ss.todo_tail = nil
		}
		ret.next, ret.prev = nil, nil
		ret.todo = false
		return ret
	} else {
		return nil
	}
}

/*
Take two input sets, in the form (x,y,mask). Munge the first by
taking either its intersection with the second or its difference
with the second. Return the new mask part of the first set.
*/
func setMunge(x1, y1 int, mask1 uint16, x2, y2 int, mask2 uint16, diff bool) uint16 {
	/*
	 * Adjust the second set so that it has the same x,y
	 * coordinates as the first.
	 */
	if absDiff(x2, x1) >= 3 || absDiff(y2, y1) >= 3 {
		mask2 = 0
	} else {
		for x2 > x1 {
			m := (^(4 | 32 | 256))
			mask2 &= uint16(m)
			mask2 <<= 1
			x2--
		}
		for x2 < x1 {
			m := ^(1 | 8 | 64)
			mask2 &= uint16(m)
			mask2 >>= 1
			x2++
		}
		for y2 > y1 {
			m := ^(64 | 128 | 256)
			mask2 &= uint16(m)
			mask2 <<= 3
			y2--
		}
		for y2 < y1 {
			m := ^(1 | 2 | 4)
			mask2 &= uint16(m)
			mask2 >>= 3
			y2++
		}
	}

	/*
	 * Invert the second set if `diff' is set (we're after A &~ B
	 * rather than A & B).
	 */
	if diff {
		mask2 ^= 511
	}

	/*
	 * Now all that's left is a logical AND.
	 */
	return mask1 & mask2
}
