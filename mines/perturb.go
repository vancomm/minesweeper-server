package mines

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"

	"github.com/sirupsen/logrus"
)

type curiosity int

const (
	verySuspicious curiosity = iota + 1
	mildlyInteresting
	boring
)

func (c curiosity) String() string {
	switch c {
	case verySuspicious:
		return "S"
	case mildlyInteresting:
		return "I"
	default:
		return "B"
	}
}

/* Structure used internally to mineperturb(). */
type square struct {
	x, y     int
	priority curiosity
	random   int32
}

func (s *square) String() string {
	return fmt.Sprintf("%d:%d(%s)", s.x, s.y, s.priority.String())
}

func squarecmp(a, b *square) int {
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

type delta int8

const (
	SetMine  delta = +1
	SetClear delta = -1
)

func (d delta) String() string {
	switch d {
	case SetMine:
		return "SetMine"
	default:
		return "SetClear"
	}
}

func (d delta) Mine() bool {
	return d == SetMine
}

/*
This is data returned from the `perturb' function. It details
which squares have become mines and which have become clear. The
solver is (of course) expected to honourably not use that
knowledge directly, but to efficently adjust its internal data
structures and proceed based on only the information it
legitimately has.
*/
type change struct {
	x, y  int
	delta delta /* +1 == become a mine; -1 == cleared */
}

func (p *change) String() string {
	return fmt.Sprintf("%s@%d:%d", p.delta.String(), p.x, p.y)
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
func (ctx *mineCtx) Perturb(
	grid *GridInfo,
	setx, sety int,
	mask word,
	r *rand.Rand,
) []*change {
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
	squares := make([]*square, 0, ctx.width*ctx.height)
	for y := range ctx.height {
		for x := range ctx.width {
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
			if (mask == 0 && (*grid)[y*ctx.width+x] == Unknown) ||
				(x >= setx && x < setx+3 &&
					y >= sety && y < sety+3 &&
					mask&(1<<((y-sety)*3+(x-setx))) != 0) {
				continue
			}

			sq := &square{x: x, y: y}

			if (*grid)[y*ctx.width+x] != Unknown {
				sq.priority = boring /* known square */
			} else {
				/*
				 * Unknown square. Examine everything around it and
				 * see if it borders on any known squares. If it
				 * does, it's class 1, otherwise it's 2.
				 */
				sq.priority = mildlyInteresting

				for dy := -1; dy <= +1; dy++ {
					for dx := -1; dx <= +1; dx++ {
						if x+dx >= 0 && x+dx < ctx.width &&
							y+dy >= 0 && y+dy < ctx.height &&
							(*grid)[(y+dy)*ctx.width+(x+dx)] != Unknown {
							sq.priority = verySuspicious
							break
						}
					}
				}
			}
			/*
			 * Finally, a random number to cause qsort to
			 * shuffle within each group.
			 */
			sq.random = r.Int32()

			squares = append(squares, sq)
		}
	}

	slices.SortFunc(squares, squarecmp)

	/*
	 * Now count up the number of full and empty squares in the set
	 * we've been provided.
	 */
	nfull, nempty := 0, 0
	if mask != 0 {
		for dy := range 3 {
			for dx := range 3 {
				if mask&(1<<(dy*3+dx)) != 0 {
					// assert(setx+dx <= ctx->w);
					// assert(sety+dy <= ctx->h);
					if setx+dx > ctx.width || sety+dy > ctx.height {
						Log.WithFields(logrus.Fields{
							"dx": dx, "dy": dy, "ctx": ctx,
						}).Fatal("out of range")
					}
					if ctx.MineAt(setx+dx, sety+dy) {
						nfull++
					} else {
						nempty++
					}
				}
			}
		}
	} else {
		for y := range ctx.height {
			for x := range ctx.width {
				if (*grid)[y*ctx.width+x] == Unknown {
					if ctx.MineAt(x, y) {
						nfull++
					} else {
						nempty++
					}
				}
			}
		}
	}

	/*
	 * Now go through our sorted list until we find either `nfull'
	 * empty squares, or `nempty' full squares; these will be
	 * swapped with the appropriate squares in the set to either
	 * fill or empty the set while keeping the same number of mines
	 * overall.
	 */
	var toFill, toEmpty []*square
	if mask != 0 {
		toFill = make([]*square, 0, 9)
		toEmpty = make([]*square, 0, 9)
	} else {
		toFill = make([]*square, 0, ctx.width*ctx.height)
		toEmpty = make([]*square, 0, ctx.width*ctx.height)
	}
	for _, sq := range squares {
		if ctx.MineAt(sq.x, sq.y) {
			toEmpty = append(toEmpty, sq)
		} else {
			toFill = append(toFill, sq)
		}
		if len(toFill) == nfull || len(toEmpty) == nempty {
			break
		}
	}

	/*
	 * If we haven't found enough empty squares outside the set to
	 * empty it into _or_ enough full squares outside it to fill it
	 * up with, we'll have to settle for doing only a partial job.
	 * In this case we choose to always _fill_ the set (because
	 * this case will tend to crop up when we're working with very
	 * high mine densities and the only way to get a solvable grid
	 * is going to be to pack most of the mines solidly around the
	 * edges). So now our job is to make a list of the empty
	 * squares in the set, and shuffle that list so that we fill a
	 * random selection of them.
	 */
	var setlist []int

	if len(toFill) != nfull && len(toEmpty) != nempty {
		if len(toEmpty) == 0 { // assert(ntoempty != 0)
			Log.WithFields(logrus.Fields{
				"toEmpty": toEmpty, "toFill": toFill,
			}).Fatal("invalid state")
		}

		setlist = make([]int, 0, ctx.width*ctx.height)

		if mask != 0 {
			for dy := range 3 {
				for dx := range 3 {
					if mask&(1<<(dy*3+dx)) != 0 {
						// assert(setx+dx <= ctx->w);
						// assert(sety+dy <= ctx->h);
						if setx+dx > ctx.width || sety+dy > ctx.height {
							Log.WithFields(logrus.Fields{
								"dx": dx, "dy": dy, "ctx": ctx,
							}).Fatal("out of range")
						}
						if !ctx.MineAt(setx+dx, sety+dy) {
							setlist = append(setlist, (sety+dy)*ctx.width+(setx+dx))
						}
					}
				}
			}
		} else {
			for y := range ctx.height {
				for x := range ctx.width {
					if (*grid)[y*ctx.width+x] == Unknown {
						if !ctx.MineAt(x, y) {
							setlist = append(setlist, y*ctx.width+x)
						}
					}
				}
			}
		}

		// assert(i > ntoempty)
		if (len(setlist)) <= len(toEmpty) {
			Log.WithFields(logrus.Fields{
				"setlist": setlist,
				"toEmpty": toEmpty,
				"toFill":  toFill,
			}).Fatal("setlist cannot be smaller than toEmpty")
		}

		/*
		 * Now pick `ntoempty' items at random from the list.
		 */
		for k := range toEmpty {
			index := k + r.IntN(len(setlist)-k)
			setlist[k], setlist[index] = setlist[index], setlist[k]
		}
	} else {
		setlist = nil
	}

	/*
	 * Now we're pretty much there. We need to either
	 * 	(a) put a mine in each of the empty squares in the set, and
	 * 	    take one out of each square in `toempty'
	 * 	(b) take a mine out of each of the full squares in the set,
	 * 	    and put one in each square in `tofill'
	 * depending on which one we've found enough squares to do.
	 *
	 * So we start by constructing our list of changes to return to
	 * the solver, so that it can update its data structures
	 * efficiently rather than having to rescan the whole grid.
	 */
	var (
		todos       []*square
		dTodo, dSet delta
	)
	if len(toFill) == nfull {
		todos = toFill
		dTodo = SetMine
		dSet = SetClear
		toEmpty = nil
	} else {
		/*
		 * (We also fall into this case if we've constructed a
		 * setlist.)
		 */
		todos = toEmpty
		dTodo = SetClear
		dSet = SetMine
		toFill = nil
	}

	changes := make([]*change, 0, 2*len(todos)) // originally snewn(2 * ntodo, struct perturbation)
	for _, t := range todos {
		changes = append(changes, &change{
			x:     t.x,
			y:     t.y,
			delta: dTodo,
		})
	}

	if setlist != nil {
		// assert(todo == toempty)
		if !reflect.DeepEqual(todos, toEmpty) {
			Log.WithFields(logrus.Fields{
				"todos": todos, "toEmpty": toEmpty,
			}).Fatal("must be equal")
		}

		for j := range toEmpty {
			changes = append(changes, &change{
				x:     setlist[j] % ctx.width,
				y:     setlist[j] / ctx.width,
				delta: dSet,
			})
		}
		setlist = nil
	} else if mask != 0 {
		for dy := range 3 {
			for dx := range 3 {
				if mask&(1<<(dy*3+dx)) != 0 {
					var currval delta
					if ctx.MineAt(setx+dx, sety+dy) {
						currval = SetMine
					} else {
						currval = SetClear
					}
					if dSet == -currval {
						changes = append(changes, &change{
							x:     setx + dx,
							y:     sety + dy,
							delta: dSet,
						})
					}
				}
			}
		}
	} else {
		for y := range ctx.height {
			for x := range ctx.width {
				if (*grid)[y*ctx.width+x] == Unknown {
					var currval delta
					if ctx.MineAt(x, y) {
						currval = SetMine
					} else {
						currval = SetClear
					}
					if dSet == -currval {
						changes = append(changes, &change{
							x:     x,
							y:     y,
							delta: dSet,
						})
					}
				}
			}
		}
	}

	// assert(i == ret->n)
	if len(changes) != 2*len(todos) {
		Log.WithFields(logrus.Fields{
			"todos": todos, "perturbs": changes,
		}).Fatal("some perturbations have not generated")
	}

	squares = nil
	todos = nil

	/*
	 * Having set up the precise list of changes we're going to
	 * make, we now simply make them and return.
	 */
	for _, c := range changes {
		var (
			x     = c.x
			y     = c.y
			delta = c.delta
		)

		/*
		 * Check we're not trying to add an existing mine or remove
		 * an absent one.
		 */
		// assert((delta < 0) ^ (ctx->grid[y*ctx->w+x] == 0))
		if delta.Mine() == ctx.MineAt(x, y) {
			Log.WithFields(logrus.Fields{
				"change": c,
				"mine":   ctx.MineAt(x, y),
			}).Fatal("trying to add an existing mine or remove an absent one")
		}

		/*
		 * Actually make the change.
		 */
		ctx.grid[y*ctx.width+x] = delta.Mine()

		/*
		 * Update any numbers already present in the grid.
		 */
		for dy := -1; dy <= +1; dy++ {
			for dx := -1; dx <= +1; dx++ {
				if x+dx >= 0 && x+dx < ctx.width &&
					y+dy >= 0 && y+dy < ctx.height &&
					(*grid)[(y+dy)*ctx.width+(x+dx)] != Unknown {
					if dx == 0 && dy == 0 {
						/*
						 * The square itself is marked as known in
						 * the grid. Mark it as a mine if it's a
						 * mine, or else work out its number.
						 */
						if delta == SetMine {
							(*grid)[y*ctx.width+x] = Mine
						} else {
							var minecount SquareInfo = 0
							for dy2 := -1; dy2 <= +1; dy2++ {
								for dx2 := -1; dx2 <= +1; dx2++ {
									if x+dx2 >= 0 && x+dx2 < ctx.width &&
										y+dy2 >= 0 && y+dy2 < ctx.height &&
										ctx.MineAt(x+dx2, y+dy2) {
										minecount++
									}
								}
							}
							(*grid)[y*ctx.width+x] = minecount
						}
					} else {
						if (*grid)[(y+dy)*ctx.width+(x+dx)] >= 0 {
							(*grid)[(y+dy)*ctx.width+(x+dx)] += SquareInfo(delta)
						}
					}
				}
			}
		}
	}

	return changes
}
