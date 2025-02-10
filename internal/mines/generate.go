package mines

import (
	"fmt"
	"math/rand/v2"
)

// panics [AssertionError]
func (p GameParams) newSolvableGrid(startX, startY int, r *rand.Rand) (grid []bool, err error) {
	width, height, mineCount, _ := p.Unpack()

	attempt := 0
	success := false // do { success = false; ... } while (!success)
	for !success {
		attempt++

		grid = make([]bool, width*height)

		/*
		 * Start by placing n mines, none of which is at x,y or within
		 * one square of it.
		 */
		{
			candidates := make([]int, 0, width*height)

			/*
			* Write down the list of possible mine locations.
			 */
			for y := range height {
				for x := range width {
					if absDiff(startY, y) > 1 || absDiff(startX, x) > 1 {
						candidates = append(candidates, y*width+x)
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
		if p.Unique {
			solveGrid := make(Grid, 0, width*height)
			ctx := &mineCtx{
				grid:  grid,
				width: width, height: height,
				sx: startX, sy: startY,
				allowBigPerturbs: attempt > 100,
			}
			prevRet := NA

			for {
				for range width * height {
					solveGrid = append(solveGrid, Unknown)
				}

				solveGrid[startY*width+startX] = ctx.Open(startX, startY)
				solveGrid[startY*width+startX] = ctx.Open(startX, startY)

				if solveGrid[startY*width+startX] != 0 {
					Log.Error("asseertion failed: mine in first square", "solveGrid", solveGrid, "ctx", ctx)
					panic(AssertionError{"mine in first square"})
				}

				solveRet := mineSolve(width, height, mineCount, solveGrid, ctx, r)
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

	if !success {
		grid = nil
		err = fmt.Errorf("could not generate a field")
	}

	return
}
