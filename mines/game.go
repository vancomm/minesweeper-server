// source: https://git.tartarus.org/simon/puzzles.git/mines.c

package mines

import "math/rand/v2"

type GameParams struct {
	Width, Height, MineCount int
	Unique                   bool
}

type MineLayout struct {
	MineGrid []bool
	/*
	 * If we haven't yet actually generated the mine layout, here's
	 * all the data we will need to do so.
	 */
	MineCount int
	Unique    bool
}

func NewLayout(params GameParams, x, y int, r *rand.Rand) (*MineLayout, error) {
	var (
		layout    *MineLayout
		grid, err = MineGen(params, x, y, r)
	)
	if err == nil {
		layout = &MineLayout{
			MineGrid:  grid,
			MineCount: params.MineCount,
			Unique:    params.Unique,
		}
	}
	return layout, err
}

type squareInfo int8

const (
	Todo        squareInfo = -10 // internal
	Question    squareInfo = -3  // ui
	Unknown     squareInfo = -2
	Mine        squareInfo = -1
	CorrectFlag squareInfo = 64 // post-game-over
	Exploded    squareInfo = 65
	WrongFlag   squareInfo = 66
	// 0-8 for empty with given number of mined neighbors
)

type (
	gridInfo []squareInfo
	openFunc func(*minectx, int, int) squareInfo
)

type GameState struct {
	GameParams
	Dead, Won, UsedSolve bool
	Layout               MineLayout /* real mine positions */
	Grid                 gridInfo   /* player knowledge */
	/*
	 * Each item in the `grid' array is one of the following values:
	 *
	 * 	- 0 to 8 mean the square is open and has a surrounding mine
	 * 	  count.
	 *
	 *  - -1 means the square is marked as a mine.
	 *
	 *  - -2 means the square is unknown.
	 *
	 * 	- -3 means the square is marked with a question mark
	 * 	  (FIXME: do we even want to bother with this?).
	 *
	 * 	- 64 means the square has had a mine revealed when the game
	 * 	  was lost.
	 *
	 * 	- 65 means the square had a mine revealed and this was the
	 * 	  one the player hits.
	 *
	 * 	- 66 means the square has a crossed-out mine because the
	 * 	  player had incorrectly marked it.
	 */
}

func (s *GameState) OpenSquare(x, y int) int {
	i := y*s.Width + x
	if s.Layout.MineGrid[i] {
		/*
		 * The player has landed on a mine. Bad luck. Expose the
		 * mine that killed them, but not the rest (in case they
		 * want to Undo and carry on playing).
		 */
		s.Dead = true
		s.Grid[i] = Exploded
		return -1
	}

	/*
	 * Otherwise, the player has opened a safe square. Mark it to-do.
	 */
	s.Grid[y*s.Width+x] = Todo /* `todo' value internal to this func */

	/*
	 * Now go through the grid finding all `todo' values and
	 * opening them. Every time one of them turns out to have no
	 * neighbouring mines, we add all its unopened neighbours to
	 * the list as well.
	 *
	 * FIXME: We really ought to be able to do this better than
	 * using repeated N^2 scans of the grid.
	 */
	for {
		doneSomething := false
		for yy := range s.Height {
			for xx := range s.Width {
				if s.Grid[yy*s.Width+xx] == Todo {
					v := 0
					for dx := -1; dx <= 1; dx++ {
						for dy := -1; dy <= 1; dy++ {
							xxx := xx + dx
							yyy := yy + dy
							if xxx >= 0 && xxx < s.Width &&
								yyy >= 0 && yyy < s.Height &&
								s.Layout.MineGrid[yyy*s.Width+xxx] {
								v++
							}
						}
					}

					s.Grid[yy*s.Width+xx] = squareInfo(v)

					if v == 0 {
						for dx := -1; dx <= 1; dx++ {
							for dy := -1; dy <= 1; dy++ {
								xxx := xx + dx
								yyy := yy + dy
								if xxx >= 0 && xxx < s.Width &&
									yyy >= 0 && yyy < s.Height &&
									s.Grid[yyy*s.Width+xxx] == Unknown {
									s.Grid[yyy*s.Width+xxx] = Todo
								}
							}
						}
					}

					doneSomething = true
				}
			}
		}

		if !doneSomething {
			break
		}
	}

	/* If the player has already lost, don't let them win as well. */
	if s.Dead {
		return 0
	}

	/*
	 * Finally, scan the grid and see if exactly as many squares
	 * are still covered as there are mines. If so, set the `won'
	 * flag and fill in mine markers on all covered squares.
	 */
	var nmines, ncovered int
	for yy := range s.Height {
		for xx := range s.Width {
			if s.Grid[yy*s.Width+xx] < 0 {
				ncovered++
			}
			if s.Layout.MineGrid[yy*s.Width+xx] {
				nmines++
			}
		}
	}

	if ncovered == nmines {
		for yy := range s.Height {
			for xx := range s.Width {
				if s.Grid[yy*s.Width+xx] < 0 {
					s.Grid[yy*s.Width+xx] = Mine
				}
			}
		}
		s.Won = true
	}

	return 0
}

func New(params GameParams, x, y int, r *rand.Rand) (*GameState, error) {
	layout, err := NewLayout(params, x, y, r)
	if err != nil {
		return nil, err
	}
	state := &GameState{
		GameParams: params,
		Layout:     *layout,
	}
	return state, err
}
