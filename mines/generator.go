// source: https://git.tartarus.org/simon/puzzles.git/mines.c

package mines

import (
	"math/rand/v2"
	"reflect"
	"slices"

	"github.com/sirupsen/logrus"
)

/* ----------------------------------------------------------------------
 * Grid generator which uses [the above] solver.
 */

type minectx struct {
	grid             []bool
	width, height    int
	sx, sy           int
	allowBigPerturbs bool
}

func (ctx minectx) mineAt(x, y int) bool {
	return ctx.grid[y*ctx.width+x]
}

// x and y must be in range of ctx.Grid's w and h
func mineOpen(ctx *minectx, x, y int) (n squareInfo) {
	if ctx.mineAt(x, y) {
		return Mine /* *bang* */
	}
	for i := -1; i <= 1; i++ {
		if x+i < 0 || x+i >= ctx.width {
			continue
		}
		for j := -1; j <= 1; j++ {
			if y+j < 0 || y+j >= ctx.height {
				continue
			}
			if i == 0 && j == 0 {
				continue
			}
			if ctx.mineAt(x+i, y+j) {
				n++
			}
		}
	}
	return n
}

type curiosity int

const (
	verySuspicious curiosity = iota + 1
	mildlyInteresting
	boring
)

/* Structure used internally to mineperturb(). */
type square struct {
	x, y     int
	priority curiosity
	random   int32
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
	r *rand.Rand,
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
			if (mask == 0 && grid[y*ctx.width+x] == Unknown) ||
				(x >= setx && (x < setx+3) &&
					y >= sety && (y < sety+3) &&
					(mask&(1<<((y-sety)*3+(x-setx)))) != 0) {
				continue
			}

			sq := &square{x: x, y: y}

			if grid[y*ctx.width+x] != Unknown {
				sq.priority = boring /* known square */
			} else {
				/*
				 * Unknown square. Examine everything around it and
				 * see if it borders on any known squares. If it
				 * does, it's class 1, otherwise it's 2.
				 */
				sq.priority = mildlyInteresting

				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if (x+dx >= 0) && (x+dx < ctx.width) &&
							(y+dy >= 0) && (y+dy < ctx.height) &&
							grid[(y+dy)*ctx.width+(x+dx)] != Unknown {
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
	var nfull, nempty int
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
					if ctx.grid[(sety+dy)*ctx.width+(setx+dx)] {
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
				if grid[y*ctx.width+x] == Unknown {
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
		if ctx.grid[sq.y*ctx.width+sq.x] {
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

		var setlist []int

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

						if !ctx.grid[(sety+dy)*ctx.width+(setx+dx)] {
							setlist = append(setlist, (sety+dy)*ctx.width+(setx+dx))
						}
					}
				}
			}
		} else {
			for y := range ctx.height {
				for x := range ctx.width {
					if grid[y*ctx.width+x] == Unknown {
						if !ctx.grid[y*ctx.width+x] {
							setlist = append(setlist, y*ctx.width+x)
						}
					}
				}
			}
		}

		// assert(i > ntoempty)
		if (len(setlist)) <= len(toEmpty) {
			Log.WithFields(logrus.Fields{
				"setlist": setlist, "toEmpty": toEmpty,
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
		dTodo, dSet perturbdelta
	)

	if len(toFill) == nfull {
		todos = toFill
		dTodo = MarkAsMine
		dSet = MarkAsClear
	} else {
		/*
		 * (We also fall into this case if we've constructed a
		 * setlist.)
		 */
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
			Log.WithFields(logrus.Fields{
				"todos": todos, "toEmpty": toEmpty,
			}).Fatal("must be equal")
		}

		for j := range toEmpty {
			perturbs = append(perturbs, &perturbation{
				x:     setlist[j] % ctx.width,
				y:     setlist[j] / ctx.width,
				delta: dSet,
			})
		}
	} else if mask != 0 {
		for dy := range 3 {
			for dx := range 3 {
				if mask&(1<<(dy*3+dx)) != 0 {
					var currval perturbdelta
					if ctx.mineAt(setx+dx, sety+dy) {
						currval = MarkAsMine
					} else {
						currval = MarkAsClear
					}
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
		for y := range ctx.height {
			for x := range ctx.width {
				if grid[y*ctx.width+x] == Unknown {
					var currval perturbdelta
					if ctx.mineAt(x, y) {
						currval = MarkAsMine
					} else {
						currval = MarkAsClear
					}
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
		Log.WithFields(logrus.Fields{
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
		if (delta < 0) == (!ctx.grid[y*ctx.width+x]) {
			Log.Fatal("trying to add an existing mine or remove an absent one")
		}

		/*
		 * Actually make the change.
		 */
		ctx.grid[y*ctx.width+x] = (delta > 0)

		/*
		 * Update any numbers already present in the grid.
		 */
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if (x+dx == 0 && x+dx < ctx.width) &&
					(y+dy >= 0 && y+dy < ctx.height) &&
					grid[(y+dy)*ctx.width+(x+dx)] != Unknown {
					if dx == 0 && dy == 0 {
						/*
						 * The square itself is marked as known in
						 * the grid. Mark it as a mine if it's a
						 * mine, or else work out its number.
						 */
						if delta == MarkAsMine {
							grid[y*ctx.width+x] = Mine
						} else {
							var minecount squareInfo
							for dy2 := -1; dy2 <= 1; dy2++ {
								for dx2 := -1; dx2 <= 1; dx2++ {
									if (x+dx2 >= 0 && x+dx2 < ctx.width) &&
										(y+dy2 >= 0 && y+dy2 < ctx.height) &&
										ctx.grid[(y+dy2)*ctx.width+(x+dx2)] {
										minecount++
									}
								}
							}
							grid[y*ctx.width+x] = minecount
						}
					} else {
						if grid[(y+dy)*ctx.width+(x+dx)] >= 0 {
							grid[(y+dy)*ctx.width+(x+dx)] += squareInfo(delta)
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

type MineGenError string

func (mge MineGenError) Error() string {
	return string(mge)
}

func MineGen(params GameParams, x, y int, r *rand.Rand) ([]bool, error) {
	var (
		width     = params.Width
		height    = params.Height
		mineCount = params.MineCount
		unique    = params.Unique
		nTries    = 0
		grid      = make([]bool, width*height)
	)

	// do { success = false; ... } while (!success)
	success := false
	for !success {
		nTries++

		/*
		 * Start by placing n mines, none of which is at x,y or within
		 * one square of it.
		 */
		{
			var mineable []int

			/*
			* Write down the list of possible mine locations.
			 */
			for i := range height {
				for j := range width {
					if absDiff(i, y) > 1 || absDiff(j, x) > 1 {
						mineable = append(mineable, i*width+j)
					}
				}
			}

			/*
			 * Now pick n off the list at random.
			 */
			k := len(mineable)
			for range mineCount {
				i := r.IntN(k)
				grid[mineable[i]] = true
				k--
				mineable[i] = mineable[k]
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
				solveGrid = make(gridInfo, width*height)
				ctx       = &minectx{
					grid:  grid,
					width: width, height: height,
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

				solveGrid[y*width+x] = mineOpen(ctx, x, y)

				// assert(solvegrid[y*w+x] == 0) /* by deliberate arrangement */
				if solveGrid[y*width+x] != 0 {
					Log.WithFields(logrus.Fields{
						"solveGrid": solveGrid, "ctx": ctx,
					}).Fatal("mine in first square")
				}

				solveRet = mineSolve(width, height, mineCount, solveGrid, mineOpen, minePerturb, ctx, r)

				if solveRet < 0 || prevRet >= 0 && solveRet >= prevRet {
					success = false
					break
				} else if solveRet == Success {
					success = true
					break
				}
			}
		} else {
			success = true
		}
	}

	var err error
	if !success {
		err = MineGenError("could not generate a field")
	}

	return grid, err
}
