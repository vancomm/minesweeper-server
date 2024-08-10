// source: https://git.tartarus.org/simon/puzzles.git/mines.c

package mines

import (
	"log"
	"math/rand/v2"

	"github.com/sirupsen/logrus"
)

var Log = logrus.New()

type GameState struct {
	GameParams
	Dead, Won, UsedSolve bool
	Grid                 []bool   /* real mine positions */
	PlayerGrid           GridInfo /* player knowledge */
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

func New(params GameParams, x, y int, r *rand.Rand) (*GameState, error) {
	grid, err := params.generate(x, y, r)
	if err != nil {
		return nil, err
	}
	playerGrid := make(GridInfo, len(grid))
	for i := range playerGrid {
		playerGrid[i] = Unknown
	}
	state := &GameState{
		GameParams: params,
		Grid:       grid,
		PlayerGrid: playerGrid,
	}
	if state.OpenSquare(x, y) != 0 {
		log.Fatalf("mine in or around starting square")
	}
	return state, err
}

func (s *GameState) OpenSquare(x, y int) int {
	i := y*s.Width + x
	if s.Grid[i] {
		/*
		 * The player has landed on a mine. Bad luck. Expose the
		 * mine that killed them, but not the rest (in case they
		 * want to Undo and carry on playing).
		 */
		s.Dead = true
		s.PlayerGrid[i] = Exploded
		return -1
	}

	/*
	 * Otherwise, the player has opened a safe square. Mark it to-do.
	 */
	s.PlayerGrid[i] = Todo /* `todo' value internal to this func */

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
				if s.PlayerGrid[yy*s.Width+xx] == Todo {
					v := 0
					for dx := -1; dx <= 1; dx++ {
						for dy := -1; dy <= 1; dy++ {
							xxx := xx + dx
							yyy := yy + dy
							if xxx >= 0 && xxx < s.Width &&
								yyy >= 0 && yyy < s.Height &&
								s.Grid[yyy*s.Width+xxx] {
								v++
							}
						}
					}

					s.PlayerGrid[yy*s.Width+xx] = SquareInfo(v)

					if v == 0 {
						for dx := -1; dx <= 1; dx++ {
							for dy := -1; dy <= 1; dy++ {
								xxx := xx + dx
								yyy := yy + dy
								if xxx >= 0 && xxx < s.Width &&
									yyy >= 0 && yyy < s.Height &&
									s.PlayerGrid[yyy*s.Width+xxx] == Unknown {
									s.PlayerGrid[yyy*s.Width+xxx] = Todo
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
			if s.PlayerGrid[yy*s.Width+xx] < 0 {
				ncovered++
			}
			if s.Grid[yy*s.Width+xx] {
				nmines++
			}
		}
	}

	if ncovered == nmines {
		for yy := range s.Height {
			for xx := range s.Width {
				if s.PlayerGrid[yy*s.Width+xx] < 0 {
					s.PlayerGrid[yy*s.Width+xx] = Mine
				}
			}
		}
		s.Won = true
	}

	return 0
}

func (s *GameState) FlagSquare(x, y int) {
	i := y*s.Width + x
	if s.PlayerGrid[i] == Unknown {
		s.PlayerGrid[i] = Mine
	} else if s.PlayerGrid[i] == Mine {
		s.PlayerGrid[i] = Unknown
	}
}

func (s *GameState) ChordSquare(x, y int) {
	i := y*s.Width + x
	if !(0 <= s.PlayerGrid[i] && s.PlayerGrid[i] <= 8) {
		return
	}
	c := int(s.PlayerGrid[i])
	js := make([]int, 8-c)
	m := 0
	for dx := -1; dx <= +1; dx++ {
		for dy := -1; dy <= +1; dy++ {
			if 0 <= x+dx && x+dx < s.Width &&
				0 <= y+dy && y+dy < s.Height &&
				(dx != 0 || dy != 0) {
				j := (y+dy)*s.Width + (x + dx)
				if s.PlayerGrid[j] == Mine {
					m++
				} else if s.PlayerGrid[j] == Unknown {
					js = append(js, j)
				}
			}
		}
	}
	if c == m {
		for _, j := range js {
			jy := j / s.Width
			jx := j % s.Width
			s.OpenSquare(jx, jy)
			if s.Dead || s.Won {
				return
			}
		}
	}
}
