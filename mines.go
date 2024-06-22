// source: https://git.tartarus.org/simon/puzzles.git/mines.c

package main

import (
	"math/rand/v2"
	"reflect"
	"slices"

	"github.com/sirupsen/logrus"
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
	sets                 *Tree234[set]
	todo_head, todo_tail *set
}

func NewSetStore() *setstore {
	return &setstore{
		sets: NewTree234(setcmp),
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
			if el, pos := ss.sets.FindRelPos(&stmp, Ge); el != nil {
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

type (
	squareInfo int8
	gridInfo   []squareInfo
	openFn     func(*minectx, int, int) squareInfo
)

const (
	Unknown squareInfo = iota - 2
	Mine
	// 0-8 for empty with given number of mined neighbors
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
	AssumeMine  perturbdelta = 1
	AssumeClear perturbdelta = -1
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
	x, y   int
	_delta perturbdelta /* +1 == become a mine; -1 == cleared */
}

type perturbcb func(ctx *minectx, grid []squareInfo, setx, sety int, mask word) []*perturbation

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
func mineSolve(
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
					newmines := s.mines - Iif(grid[i] == Mine, 1, 0)
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
				if p._delta < 0 && grid[p.y*w+p.x] != Unknown {
					std.add(p.y*w + p.x)
				}
			}
		}
	}
	return
}

/* ----------------------------------------------------------------------
 * Grid generator which uses the above solver.
 */

type minectx struct {
	grid             []bool
	w, h             int
	sx, sy           int
	allowBigPerturbs bool
}

func (ctx minectx) at(x, y int) bool {
	return ctx.grid[y*ctx.w+x]
}

// x and y must be in range of ctx.Grid's w and h
func mineOpen(ctx *minectx, x, y int) (n squareInfo) {
	if ctx.at(x, y) {
		return Mine /* *bang* */
	}
	for i := -1; i <= 1; i++ {
		if x+i < 0 || x+i >= ctx.w {
			continue
		}
		for j := -1; j <= 1; j++ {
			if y+j < 0 || y+j >= ctx.h {
				continue
			}
			if i == 0 && j == 0 {
				continue
			}
			if ctx.at(x+i, y+j) {
				n++
			}
		}
	}
	return n
}

type curiosity int

const (
	VerySuspicious curiosity = iota + 1
	MildlyInteresting
	Boring
)

/* Structure used internally to mineperturb(). */
type square struct {
	x, y     int
	priority curiosity
	random   int
}

func squarecmp(a, b square) int {
	if a.priority < b.priority {
		return -1
	}
	if a.priority > b.priority {
		return 1
	}
	if a.random < b.random {
		return -1
	}
	if a.random > b.random {
		return 1
	}
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
	return 0
}

func absDiff(x, y int) int {
	if x > y {
		return x - y
	}
	return y - x
}

/*
Normally this function is passed an (x,y,mask) set description.
On occasions, though, there is no _localised_ set being used,
and the set being perturbed is supposed to be the entirety of
the unreachable area. This is signified by the special case
mask==0: in this case, anything labelled -2 in the grid is part
of the set.

Allowing perturbation in this special case appears to make it
guaranteeably possible to generate a workable grid for any mine
density, but they tend to be a bit boring, with mines packed
densely into far corners of the grid and the remainder being
less dense than one might like. Therefore, to improve overall
grid quality I disable this feature for the first few attempts,
and fall back to it after no useful grid has been generated.
*/
func minePerturb(
	ctx *minectx,
	grid []squareInfo,
	setx, sety int,
	mask word,
) []*perturbation {
	if mask == 0 && !ctx.allowBigPerturbs {
		return nil
	}

	/*
	* Make a list of all the squares in the grid which we can
	* possibly use. This list should be in preference order, which
	* means
	*
	*  - first, unknown squares on the boundary of known space
	*  - next, unknown squares beyond that boundary
	* 	- as a very last resort, known squares, but not within one
	* 	  square of the starting position.
	*
	* Each of these sections needs to be shuffled independently.
	* We do this by preparing list of all squares and then sorting
	* it with a random secondary key.
	 */
	var (
		squares = make([]square, ctx.w*ctx.h)
		n       = 0
	)
	for y := range ctx.h {
		for x := range ctx.w {
			/*
			 * If this square is too near the starting position,
			 * don't put it on the list at all.
			 */
			if absDiff(y, ctx.sy) <= 1 && absDiff(x, ctx.sx) <= 1 {
				continue
			}

			/*
			 * If this square is in the input set, also don't put
			 * it on the list!
			 */
			if (mask == 0 && grid[y*ctx.w+x] == Unknown) ||
				(x >= setx && (x < setx+3) &&
					y >= sety && (y < sety+3) &&
					(mask&(1<<((y-sety)*3+(x-setx)))) != 0) {
				continue
			}

			squares[n].x = x
			squares[n].y = y

			if grid[y*ctx.w+x] != Unknown {
				squares[n].priority = Boring // known square
			} else {
				/*
					Unknown square. Examine everything around it and see if it
					borders on any known squares. If it does, it's class 1,
					otherwise it's 2.
				*/
				squares[n].priority = MildlyInteresting

				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if x+dx >= 0 && x+dx < ctx.w &&
							y+dy >= 0 && y+dy < ctx.h &&
							grid[(y+dy)*ctx.w+(x+dx)] != Unknown {
							squares[n].priority = VerySuspicious
							break
						}
					}
				}
			}
			squares[n].random = int(rand.Uint32())
			n++
		}
	}

	slices.SortFunc(squares, squarecmp)

	var nfull, nempty int
	if mask != 0 {
		for dy := 0; dy < 3; dy++ {
			for dx := 0; dx < 3; dx++ {
				if mask&(1<<(dy*3+dx)) != 0 {
					if setx+dx > ctx.w || sety+dy > ctx.h {
						log.WithFields(logrus.Fields{
							"dx": dx, "dy": dy,
						}).Fatal("out of range")
					}

					if ctx.grid[(sety+dy)*ctx.w+(setx+dx)] {
						nfull++
					} else {
						nempty++
					}
				}
			}
		}
	} else {
		for y := 0; y < ctx.h; y++ {
			for x := 0; x < ctx.w; x++ {
				if grid[y*ctx.w+x] == Unknown {
					nfull++
				} else {
					nempty++
				}
			}
		}
	}

	var (
		// window          = Iif(mask != 0, 9, ctx.w*ctx.h)
		toFill, toEmpty []*square
	)
	// for range window {
	// 	tofill = append(tofill, &square{})
	// 	toempty = append(toempty, &square{})
	// }
	for i := 0; i < n; i++ {
		sq := squares[i]
		if ctx.grid[sq.y*ctx.w+sq.x] {
			toEmpty = append(toEmpty, &sq)
		} else {
			toFill = append(toFill, &sq)
		}
		if len(toFill) == nfull || len(toEmpty) == nempty {
			break
		}
	}

	var setlist []int
	if len(toFill) != nfull && len(toEmpty) != nempty {
		if len(toEmpty) == 0 {
			log.Fatal("len(toEmpty) is 0 when it must not be")
		}
		setlist = make([]int, ctx.w*ctx.h)
		i := 0
		if mask != 0 {
			for dy := 0; dy < 3; dy++ {
				for dx := 0; dx < 3; dx++ {
					if mask&(1<<(dy*3+dx)) != 0 {
						if setx+dx > ctx.w || sety+dy > ctx.h {
							log.WithFields(logrus.Fields{
								"dx": dx, "dy": dy,
							}).Fatal("out of range")
						}

						if !ctx.grid[(sety+dy)*ctx.w+(setx+dx)] {
							setlist[i] = (sety+dy)*ctx.w + (setx + dx)
							i++
						}
					}
				}
			}
		} else {
			for y := 0; y < ctx.h; y++ {
				for x := 0; x < ctx.w; x++ {
					if grid[y*ctx.w+x] == Unknown {
						if !ctx.grid[y*ctx.w+x] {
							setlist[i] = y*ctx.w + x
							i++
						}
					}
				}
			}
		}

		if i <= len(toEmpty) {
			log.WithFields(logrus.Fields{
				"i": i, "len(toEmpty)": len(toEmpty),
			}).Fatal("i must be less than len(toEmpty)")
		}

		for k := 0; k < len(toEmpty); k++ {
			index := k + rand.IntN(i-k)
			setlist[k], setlist[index] = setlist[index], setlist[k]
		}
	} else {
		setlist = nil
	}

	var (
		todos       []*square
		dTodo, dSet perturbdelta
	)
	if len(toFill) == nfull {
		todos = toFill
		dTodo = AssumeMine
		dSet = AssumeClear
	} else {
		todos = toEmpty
		dTodo = AssumeClear
		dSet = AssumeMine
	}

	var perturbs []*perturbation // originally changes with len = 2 * len(todos)

	for _, t := range todos {
		perturbs = append(perturbs, &perturbation{
			x:      t.x,
			y:      t.y,
			_delta: dTodo,
		})
	}

	if setlist != nil {
		if !reflect.DeepEqual(todos, toEmpty) {
			log.WithFields(logrus.Fields{
				"todo": todos, "toempty": toEmpty,
			}).Fatal("todo must be ")
		}
		for j := range len(toEmpty) {
			perturbs = append(perturbs, &perturbation{
				x:      setlist[j] % ctx.w,
				y:      setlist[j] / ctx.w,
				_delta: dSet,
			})
		}
	} else if mask != 0 {
		for dy := range 3 {
			for dx := range 3 {
				if mask&(1<<(dy*3+dx)) != 0 {
					currval := Iif(ctx.grid[(sety+dy)*ctx.w+(setx+dx)], AssumeMine, AssumeClear)
					if dSet == -currval {
						perturbs = append(perturbs, &perturbation{
							x:      setx + dx,
							y:      sety + dy,
							_delta: dSet,
						})
					}
				}
			}
		}
	} else {
		for y := range ctx.h {
			for x := range ctx.w {
				if grid[y*ctx.w+x] == Unknown {
					currval := Iif(ctx.grid[y*ctx.w+x], AssumeMine, AssumeClear)
					if dSet == -currval {
						perturbs = append(perturbs, &perturbation{
							x:      x,
							y:      y,
							_delta: dSet,
						})
					}
				}
			}
		}
	}

	if len(perturbs) != 2*len(todos) { // assert
		log.WithFields(logrus.Fields{
			"todos": len(todos), "perturbs": len(perturbs),
		}).Fatal("some perturbations have not generated")
	}

	squares = nil
	todos = nil

	for _, p := range perturbs {
		var (
			x     = p.x
			y     = p.y
			delta = p._delta
		)

		if (delta < 0) == (!ctx.grid[y*ctx.w+x]) { // assert
			log.Fatal("trying to add an existing mine or remove an absent one")
		}

		ctx.grid[y*ctx.w+x] = (delta > 0)

		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if x+dx == 0 && x+dx < ctx.w &&
					y+dy >= 0 && y+dy < ctx.h &&
					grid[(y+dy)*ctx.w+(x+dx)] != Unknown {
					if dx == 0 && dy == 0 {
						if delta > 0 {
							grid[y*ctx.w+x] = Mine
						} else {
							var minecount squareInfo
							for dy2 := -1; dy2 <= 1; dy2++ {
								for dx2 := -1; dx2 <= 1; dx2++ {
									if x+dx2 >= 0 && x+dx2 < ctx.w &&
										y+dy2 >= 0 && y+dy2 < ctx.h &&
										ctx.grid[(y+dy2)*ctx.w+(x+dx2)] {
										minecount++
									}
								}
							}
							grid[y*ctx.w+x] = minecount
						}
					} else {
						if grid[(y+dy)*ctx.w+(x+dx)] >= 0 {
							grid[(y+dy)*ctx.w+(x+dx)] += squareInfo(delta)
						}
					}
				}
			}
		}
	}
	return perturbs
}

type SolveResult int8

const (
	Stalled SolveResult = iota - 1
	Success
	// values >0 mean given number of perturbations was required
)
