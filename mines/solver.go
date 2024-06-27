// source: https://git.tartarus.org/simon/puzzles.git/mines.c

package mines

import (
	"math/rand/v2"

	"github.com/vancomm/minesweeper-server/tree234"
)

/* ----------------------------------------------------------------------
 * Minesweeper solver, used to ensure the generated grids are
 * solvable without having to take risks.
 */

type word uint16

/*
Count the bits in a word. Only needs to cope with 16 bits.
*/
func (word word) bitcount() int {
	word = ((word & 0xAAAA) >> 1) + (word & 0x5555)
	word = ((word & 0xCCCC) >> 2) + (word & 0x3333)
	word = ((word & 0xF0F0) >> 4) + (word & 0x0F0F)
	word = ((word & 0xFF00) >> 8) + (word & 0x00FF)
	return int(word)
}

/*
We use a tree234 to store a large number of small localised
sets, each with a mine count. We also keep some of those sets
linked together into a to-do list.
*/
type set struct {
	x, y       int
	mask       word
	mines      int
	todo       bool
	next, prev *set
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

func NewSetStore() *setstore {
	return &setstore{
		sets: tree234.New(setcmp),
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
		ss.todo_tail = s
		s.next = nil
		s.todo = true
	}
}

func (ss *setstore) add(x, y int, mask word, mines int) {
	if mask == 0 { // assert mask != 0
		log.Fatal("mask cannot be 0")
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
		return
	}

	/*
	 * We've added a new set to the tree, so put it on the todo
	 * list.
	 */
	ss.addTodo(s)
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
func (ss *setstore) overlap(x, y int, mask word) (ret []*set) {
	for xx := x - 3; xx < x+3; xx++ {
		for yy := y - 3; yy < y+3; yy++ {
			/*
			 * Find the first set with these top left coordinates.
			 */
			stmp := set{
				x:    x,
				y:    y,
				mask: 0,
			}
			if el, pos := ss.sets.FindRelPos(&stmp, tree234.Ge); el != nil {
				for s := ss.sets.Index(pos); s != nil &&
					s.x == xx && s.y == yy; {
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
					pos++
				}
			}
		}
	}
	return
}

/*
Get an element from the head of the set todo list.
*/
func (ss *setstore) todo() (ret *set) {
	if ret = ss.todo_head; ret != nil {
		ss.todo_head = ret.next
		if ss.todo_head != nil {
			ss.todo_head.prev = nil
		} else {
			ss.todo_tail = nil
		}
		ret.next, ret.prev = nil, nil
		ret.todo = false
	}
	return
}

/*
Take two input sets, in the form (x,y,mask). Munge the first by
taking either its intersection with the second or its difference
with the second. Return the new mask part of the first set.
*/
func setMunge(
	x1, y1 int, mask1 word, x2, y2 int, mask2 word, diff bool,
) word {
	/*
	 * Adjust the second set so that it has the same x,y
	 * coordinates as the first.
	 */
	if absDiff(x2, x1) >= 3 || absDiff(y2, y1) >= 3 {
		mask2 = 0
	} else {
		for x2 > x1 {
			m := ^(4 | 32 | 256)
			mask2 &= word(m)
			mask2 <<= 1
			x2--
		}
		for x2 < x1 {
			m := ^(1 | 8 | 64)
			mask2 &= word(m)
			mask2 >>= 1
			x2++
		}
		for y2 > y1 {
			m := ^(64 | 128 | 256)
			mask2 &= word(m)
			mask2 <<= 3
			y2--
		}
		for y2 < y1 {
			m := ^(1 | 2 | 4)
			mask2 &= word(m)
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

type squaretodo struct {
	next       []int
	head, tail int
}

func (std *squaretodo) add(i int) {
	if std.tail >= 0 {
		std.next[std.tail] = i
	} else {
		std.head = i
	}
	std.tail = i
	std.next[i] = -1
}

type squareInfo int8

const (
	Unknown     squareInfo = -2
	Mine        squareInfo = -1
	CorrectFlag squareInfo = 64
	Exploded    squareInfo = 65
	WrongFlag   squareInfo = 66
	// 0-8 for empty with given number of mined neighbors
)

type (
	gridInfo []squareInfo
	openFn   func(*minectx, int, int) squareInfo
)

func (grid *gridInfo) knownSquares(
	w int,
	std *squaretodo,
	open openFn, openctx *minectx,
	x, y int, mask word, mine bool,
) {
	var bit word = 1
	for yy := range 3 {
		for xx := range 3 {
			if mask&bit != 0 {
				i := (y+yy)*w + (x + xx)
				/*
				 * It's possible that this square is _already_
				 * known, in which case we don't try to add it to
				 * the list twice.
				 */
				if (*grid)[i] == Unknown {
					if mine {
						(*grid)[i] = Mine /* and don't open it! */
					} else {
						(*grid)[i] = open(openctx, x+xx, y+yy) /* *bang* */

						if (*grid)[i] == Mine { // assert grid[i] != -1
							log.Fatal("boom")
						}
					}
					std.add(i)
				}
			}
			bit <<= 1
		}
	}
}

type perturbdelta int8

const (
	MarkAsMine  perturbdelta = 1
	MarkAsClear perturbdelta = -1
)

/*
This is data returned from the `perturb' function. It details
which squares have become mines and which have become clear. The
solver is (of course) expected to honourably not use that
knowledge directly, but to efficently adjust its internal data
structures and proceed based on only the information it
legitimately has.
*/
type perturbation struct {
	x, y  int
	delta perturbdelta /* +1 == become a mine; -1 == cleared */
}

type perturbcb func(ctx *minectx, grid gridInfo, setx, sety int, mask word) []*perturbation

/*
Main solver entry point. You give it a grid of existing
knowledge (-1 for a square known to be a mine, 0-8 for empty
squares with a given number of neighbours, -2 for completely
unknown), plus a function which you can call to open new squares
once you're confident of them. It fills in as much more of the
grid as it can.

Return value is:

  - -1 means deduction stalled and nothing could be done
  - 0 means deduction succeeded fully
  - '>0' means deduction succeeded but some number of perturbation
    steps were required; the exact return value is the number of
    perturb calls.
*/
func MineSolve(
	w, h, n int,
	grid gridInfo,
	open openFn,
	perturb perturbcb,
	ctx *minectx,
) (res SolveResult) {
	var (
		ss        = NewSetStore()
		std       = &squaretodo{}
		nperturbs = 0
	)

	std.next = make([]int, w*h)
	std.head, std.tail = -1, -1

	for y := range h {
		for x := range w {
			i := y*w + x
			if grid[i] != Unknown {
				std.add(i)
			}
		}
	}

	for {
		doneSomething := false

		for std.head != -1 {
			i := std.head
			std.head = std.next[i]
			if std.head == -1 {
				std.tail = -1
			}
			x, y := i%w, i/w
			if mines := grid[i]; mines >= 0 {
				var (
					bit word = 1
					val word = 0
				)
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if x+dx < 0 || x+dx >= w || y+dy < 0 || y+dy >= h {
							// do nothing
						} else if grid[i+dy*w+dx] == Mine {
							mines--
						} else if grid[i+dy*w+dx] == Unknown {
							val |= bit
						}
						bit <<= 1
					}
				}
				if val != 0 {
					ss.add(x-1, y-1, val, int(mines))
				}
			}
			{
				for _, s := range ss.overlap(x, y, 1) {
					newmask := setMunge(s.x, s.y, s.mask, x, y, 1, true)
					newmines := s.mines - iif(grid[i] == Mine, 1, 0)
					if newmask != 0 {
						ss.add(s.x, s.y, newmask, newmines)
					}
					ss.remove(s)
				}
			}
			doneSomething = true
		}

		if s := ss.todo(); s != nil {
			if s.mines == 0 || s.mines == s.mask.bitcount() {
				grid.knownSquares(w, std, open, ctx, s.x, s.y, s.mask, s.mines != 0)
				continue
			}
			for _, s2 := range ss.overlap(s.x, s.y, s.mask) {
				swing := setMunge(s.x, s.y, s.mask, s2.x, s2.y, s2.mask, true)
				s2wing := setMunge(s2.x, s2.y, s2.mask, s.x, s.y, s.mask, true)
				swc := swing.bitcount()
				s2wc := s2wing.bitcount()

				if (swc == s.mines-s2.mines) || (s2wc == s2.mines-s.mines) {
					grid.knownSquares(w, std, open, ctx,
						s.x, s.y, swing,
						(swc == s.mines-s2.mines))
					grid.knownSquares(w, std, open, ctx,
						s2.x, s2.y, s2wing,
						(s2wc == s2.mines-s.mines))
					continue
				}

				if swc == 0 && s2wc != 0 {
					ss.add(s2.x, s2.y, s2wing, s2.mines-s.mines)
				} else if s2wc == 0 && swc != 0 {
					ss.add(s.x, s.y, swing, s.mines-s2.mines)
				}
			}
			doneSomething = true
		} else if n >= 0 {
			/*
				Global deduction
			*/

			squaresleft := 0
			minesleft := n
			for i := range w * h {
				if grid[i] == Mine {
					minesleft--
				} else if grid[i] == Unknown {
					squaresleft++
				}
			}

			if squaresleft == 0 {
				break
			}

			if minesleft == 0 || minesleft == squaresleft {
				for i := range w * h {
					if grid[i] == Unknown {
						grid.knownSquares(w, std, open, ctx,
							i%w, i/w, 1, minesleft != 0)
					}
				}
				continue
			}

			setused := make([]bool, 10)
			nsets := ss.sets.Count()

			if nsets <= len(setused) {
				var sets []*set
				for i := range nsets {
					sets = append(sets, ss.sets.Index(i))
				}
				cursor := 0
				for {
					if cursor < nsets {
						ok := true
						for i := range cursor {
							if setused[i] && setMunge(
								sets[cursor].x, sets[cursor].y, sets[cursor].mask,
								sets[i].x, sets[i].y, sets[i].mask, false,
							) != 0 {
								ok = false
								break
							}
						}
						if ok {
							minesleft -= sets[cursor].mines
							squaresleft -= sets[cursor].mask.bitcount()
						}
						setused[cursor] = ok
						cursor++
					} else {
						if squaresleft > 0 && (minesleft == 0 || minesleft == squaresleft) {
							for i := range w * h {
								if grid[i] == Unknown {
									outside := true
									y := i / w
									x := i % w
									for j := range nsets {
										if setused[j] &&
											setMunge(
												sets[j].x, sets[j].y, sets[j].mask,
												x, y, 1, false,
											) != 0 {
											outside = false
											break
										}
									}
									if outside {
										grid.knownSquares(
											w, std, open, ctx,
											x, y, 1, minesleft != 0,
										)
									}
								}
							}
							doneSomething = true
							break
						}
						cursor--
						for cursor >= 0 && !setused[cursor] {
							cursor--
						}
						if cursor >= 0 {
							minesleft += sets[cursor].mines
							squaresleft += sets[cursor].mask.bitcount()
							setused[cursor] = false
							cursor++
						} else {
							break
						}
					}
				}
			}
		}

		if doneSomething {
			continue
		}

		nperturbs++
		var ret []*perturbation
		if c := ss.sets.Count(); c == 0 {
			ret = perturb(ctx, grid, 0, 0, 0)
		} else {
			s := ss.sets.Index(rand.IntN(c))
			ret = perturb(ctx, grid, s.x, s.y, s.mask)
		}
		if len(ret) > 0 {
			for _, p := range ret {
				if p.delta < 0 && grid[p.y*w+p.x] != Unknown {
					std.add(p.y*w + p.x)
				}
			}
		}
	}
	return
}
