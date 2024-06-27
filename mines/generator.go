// source: https://git.tartarus.org/simon/puzzles.git/mines.c

package mines

import (
	"math/rand/v2"
	"reflect"
	"slices"

	"github.com/sirupsen/logrus"
)

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
	grid gridInfo,
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
	var squares []*square
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

			sq := &square{x: x, y: y}

			if grid[y*ctx.w+x] != Unknown {
				sq.priority = Boring /* known square */
			} else {
				/*
				 * Unknown square. Examine everything around it and
				 * see if it borders on any known squares. If it
				 * does, it's class 1, otherwise it's 2.
				 */
				sq.priority = MildlyInteresting

				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if x+dx >= 0 && x+dx < ctx.w &&
							y+dy >= 0 && y+dy < ctx.h &&
							grid[(y+dy)*ctx.w+(x+dx)] != Unknown {
							sq.priority = VerySuspicious
							break
						}
					}
				}
			}
			/*
			 * Finally, a random number to cause qsort to
			 * shuffle within each group.
			 */
			squares = append(squares, sq)
		}
	}

	slices.SortFunc(squares, squarecmp)

	/*
	 * Now count up the number of full and empty squares in the set
	 * we've been provided.
	 */
	var nfull, nempty int
	if mask != 0 {
		for dy := range 3 {
			for dx := range 3 {
				if mask&(1<<(dy*3+dx)) != 0 {
					// assert(setx+dx <= ctx->w);
					// assert(sety+dy <= ctx->h);
					if setx+dx > ctx.w || sety+dy > ctx.h {
						log.WithFields(logrus.Fields{
							"dx": dx, "dy": dy, "ctx": ctx,
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
		for y := range ctx.h {
			for x := range ctx.w {
				if grid[y*ctx.w+x] == Unknown {
					nfull++
				} else {
					nempty++
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
	var (
		toFill, toEmpty []*square
	)
	for _, sq := range squares {
		if ctx.grid[sq.y*ctx.w+sq.x] {
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
			log.WithFields(logrus.Fields{
				"toEmpty": toEmpty, "toFill": toFill,
			}).Fatal("invalid state")
		}
		setlist = make([]int, ctx.w*ctx.h)
		i := 0
		if mask != 0 {
			for dy := range 3 {
				for dx := range 3 {
					if mask&(1<<(dy*3+dx)) != 0 {
						// assert(setx+dx <= ctx->w);
						// assert(sety+dy <= ctx->h);
						if setx+dx > ctx.w || sety+dy > ctx.h {
							log.WithFields(logrus.Fields{
								"dx": dx, "dy": dy, "ctx": ctx,
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
			for y := range ctx.h {
				for x := range ctx.w {
					if grid[y*ctx.w+x] == Unknown {
						if !ctx.grid[y*ctx.w+x] {
							setlist[i] = y*ctx.w + x
							i++
						}
					}
				}
			}
		}

		// assert(i > ntoempty)
		if i <= len(toEmpty) {
			log.WithFields(logrus.Fields{
				"i": i, "toEmpty": toEmpty,
			}).Fatal("i must be less than len(toEmpty)")
		}

		/*
		 * Now pick `ntoempty' items at random from the list.
		 */
		for k := range toEmpty {
			index := k + rand.IntN(i-k)
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
		dTodo, dSet perturbdelta
	)

	if len(toFill) == nfull {
		todos = toFill
		dTodo = MarkAsMine
		dSet = MarkAsClear
	} else {
		todos = toEmpty
		dTodo = MarkAsClear
		dSet = MarkAsMine
	}

	var perturbs []*perturbation // originally snewn(2 * ntodo, struct perturbation)

	for _, t := range todos {
		perturbs = append(perturbs, &perturbation{
			x:     t.x,
			y:     t.y,
			delta: dTodo,
		})
	}

	if setlist != nil {
		// assert(todo == toempty)
		if !reflect.DeepEqual(todos, toEmpty) {
			log.WithFields(logrus.Fields{
				"todos": todos, "toEmpty": toEmpty,
			}).Fatal("must be equal")
		}

		for j := range toEmpty {
			perturbs = append(perturbs, &perturbation{
				x:     setlist[j] % ctx.w,
				y:     setlist[j] / ctx.w,
				delta: dSet,
			})
		}
	} else if mask != 0 {
		for dy := range 3 {
			for dx := range 3 {
				if mask&(1<<(dy*3+dx)) != 0 {
					currval := iif(ctx.grid[(sety+dy)*ctx.w+(setx+dx)], MarkAsMine, MarkAsClear)
					if dSet == -currval {
						perturbs = append(perturbs, &perturbation{
							x:     setx + dx,
							y:     sety + dy,
							delta: dSet,
						})
					}
				}
			}
		}
	} else {
		for y := range ctx.h {
			for x := range ctx.w {
				if grid[y*ctx.w+x] == Unknown {
					currval := iif(ctx.grid[y*ctx.w+x], MarkAsMine, MarkAsClear)
					if dSet == -currval {
						perturbs = append(perturbs, &perturbation{
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
	if len(perturbs) != 2*len(todos) {
		log.WithFields(logrus.Fields{
			"todos": todos, "perturbs": perturbs,
		}).Fatal("some perturbations have not generated")
	}

	/*
	 * Having set up the precise list of changes we're going to
	 * make, we now simply make them and return.
	 */
	for _, p := range perturbs {
		var (
			x     = p.x
			y     = p.y
			delta = p.delta
		)

		/*
		 * Check we're not trying to add an existing mine or remove
		 * an absent one.
		 */
		// assert((delta < 0) ^ (ctx->grid[y*ctx->w+x] == 0))
		if (delta < 0) == (!ctx.grid[y*ctx.w+x]) {
			log.Fatal("trying to add an existing mine or remove an absent one")
		}

		/*
		 * Actually make the change.
		 */
		ctx.grid[y*ctx.w+x] = (delta > 0)

		/*
		 * Update any numbers already present in the grid.
		 */
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if (x+dx == 0 && x+dx < ctx.w) &&
					(y+dy >= 0 && y+dy < ctx.h) &&
					grid[(y+dy)*ctx.w+(x+dx)] != Unknown {
					if dx == 0 && dy == 0 {
						/*
						 * The square itself is marked as known in
						 * the grid. Mark it as a mine if it's a
						 * mine, or else work out its number.
						 */
						if delta == MarkAsMine {
							grid[y*ctx.w+x] = Mine
						} else {
							var minecount squareInfo
							for dy2 := -1; dy2 <= 1; dy2++ {
								for dx2 := -1; dx2 <= 1; dx2++ {
									if (x+dx2 >= 0 && x+dx2 < ctx.w) &&
										(y+dy2 >= 0 && y+dy2 < ctx.h) &&
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
	NA SolveResult = iota - 2
	Stalled
	Success
	// values >0 mean given number of perturbations was required
)

func MineGen(params GameParams, x, y int) []bool {
	var (
		w       = params.Width
		h       = params.Height
		n       = params.MineCount
		unique  = params.Unique
		success bool
		nTries  uint64
		ret     = make([]bool, w*h)
	)

	for !success {
		success = false
		nTries++

		/*
		 * Start by placing n mines, none of which is at x,y or within
		 * one square of it.
		 */
		{
			var tmp []int

			/*
			* Write down the list of possible mine locations.
			 */
			for i := range h {
				for j := range w {
					if absDiff(i, y) > 1 || absDiff(j, x) > 1 {
						tmp = append(tmp, i*w+j)
					}
				}
			}

			/*
			 * Now pick n off the list at random.
			 */
			k := len(tmp)
			for nn := n; nn > 0; n-- {
				i := rand.IntN(k)
				ret[tmp[i]] = true
				k--
				tmp[i] = tmp[k]
			}
		}

		/*
		 * Now set up a results grid to run the solver in, and a
		 * context for the solver to open squares. Then run the solver
		 * repeatedly; if the number of perturb steps ever goes up or
		 * it ever returns -1, give up completely.
		 *
		 * We bypass this bit if we're not after a unique grid.
		 */
		if unique {
			var (
				solveGrid = make(gridInfo, w*h)
				ctx       = &minectx{
					grid: ret,
					w:    w, h: w,
					sx: x, sy: y,
					allowBigPerturbs: nTries > 100,
				}
				solveRet SolveResult
				prevRet  = NA
			)

			for {
				for i := range solveGrid {
					solveGrid[i] = Unknown
				}

				solveGrid[y*w+x] = mineOpen(ctx, x, y)

				// assert(solvegrid[y*w+x] == 0) /* by deliberate arrangement */
				if solveGrid[y*w+x] != 0 {
					log.WithFields(logrus.Fields{
						"solveGrid": solveGrid, "ctx": ctx,
					}).Fatal("mine in first square")
				}

				solveRet = MineSolve(w, h, n, solveGrid, mineOpen, minePerturb, ctx)

				if solveRet < 0 || prevRet >= 0 && solveRet >= prevRet {
					success = false
					break
				} else if solveRet == 0 {
					success = true
				}
			}
		} else {
			success = true
		}
	}
	return ret
}
