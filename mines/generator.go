// source: https://git.tartarus.org/simon/puzzles.git/mines.c

package mines

import (
	"fmt"
	"math/rand/v2"

	"github.com/sirupsen/logrus"
)

/* ----------------------------------------------------------------------
 * Grid generator which uses the [above] solver.
 */

type GameParams struct {
	Width, Height, MineCount int
	Unique                   bool
}

func (params GameParams) generate(x, y int, r *rand.Rand) ([]bool, error) {
	var (
		w         = params.Width
		h         = params.Height
		mineCount = params.MineCount
		nTries    = 0
		grid      []bool
	)

	// do { success = false; ... } while (!success)
	success := false
	for !success {
		nTries++
		grid = make([]bool, w*h)

		/*
		 * Start by placing n mines, none of which is at x,y or within
		 * one square of it.
		 */
		{
			candidates := make([]int, 0, w*h)

			/*
			* Write down the list of possible mine locations.
			 */
			for i := range h {
				for j := range w {
					if absDiff(i, y) > 1 || absDiff(j, x) > 1 {
						candidates = append(candidates, i*w+j)
					}
				}
			}

			/*
			 * Now pick n off the list at random.
			 */
			k := len(candidates)
			for range mineCount {
				i := r.IntN(k)
				grid[candidates[i]] = true
				k--
				candidates[i] = candidates[k]
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
		if params.Unique {
			var (
				solveGrid = make(GridInfo, 0, w*h)
				ctx       = &mineCtx{
					grid:  grid,
					width: w, height: h,
					sx: x, sy: y,
					allowBigPerturbs: nTries > 100,
				}
				prevRet = NA
			)

			for {
				for range w * h {
					solveGrid = append(solveGrid, Unknown)
				}

				solveGrid[y*w+x] = ctx.Open(x, y)

				if solveGrid[y*w+x] != 0 {
					Log.WithFields(logrus.Fields{
						"solveGrid": solveGrid, "ctx": ctx,
					}).Fatal("mine in first square")
				}

				solveRet := mineSolve(w, h, mineCount, solveGrid, ctx, r)
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
		err = fmt.Errorf("could not generate a field")
	}

	return grid, err
}
