// source: https://git.tartarus.org/simon/puzzles.git/mines.c

package mines

import (
	"bytes"
	"encoding/gob"
	"errors"
	"log/slog"
	"math/rand/v2"
)

var Log *slog.Logger = slog.Default()

type GameState struct {
	Dead, Won, UsedSolve bool
	Grid                 []bool /* real mine points */
	PlayerGrid           Grid   /* player knowledge */
	GameParams
}

func DecodeGameState(buf []byte) (*GameState, error) {
	var game GameState
	err := gob.NewDecoder(bytes.NewBuffer(buf)).Decode(&game)
	if err != nil {
		return nil, err
	}
	return &game, err
}

func (g GameState) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(g)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func NewGame(params *GameParams, x, y int, r *rand.Rand) (state *GameState, err error) {
	defer func() {
		var ae AssertionError
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				if errors.As(e, &ae) {
					state, err = nil, ae
				}
			}
		}
	}()

	grid, err := params.newSolvableGrid(x, y, r)
	if err != nil {
		return nil, err
	}
	playerGrid := make(Grid, len(grid))
	for i := range playerGrid {
		playerGrid[i] = Unknown
	}
	state = &GameState{
		GameParams: *params,
		Grid:       grid,
		PlayerGrid: playerGrid,
	}
	if state.OpenCell(x, y) != 0 {
		return nil, AssertionError{"mine in starting cell"}
	}
	return state, err
}

func (s *GameState) OpenCell(x, y int) int {
	i := y*s.Width + x
	if s.Grid[i] {
		/*
		 * The player has landed on a mine. Bad luck. Expose the
		 * mine that killed them, but not the rest (in case they
		 * want to Undo and carry on playing).
		 */
		s.Dead = true
		s.PlayerGrid[i] = ExplodedMine
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
					s.PlayerGrid[yy*s.Width+xx] = CellState(v)
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
				if s.PlayerGrid[yy*s.Width+xx] == -2 {
					s.PlayerGrid[yy*s.Width+xx] = UnflaggedMine
				}
			}
		}
		s.Won = true
	}

	return 0
}

func (s *GameState) FlagCell(x, y int) {
	i := y*s.Width + x
	if s.PlayerGrid[i] == Unknown {
		s.PlayerGrid[i] = Flagged
	} else if s.PlayerGrid[i] == Flagged {
		s.PlayerGrid[i] = Unknown
	}
}

func (s *GameState) ChordCell(x, y int) {
	i := y*s.Width + x
	if !(0 <= s.PlayerGrid[i] && s.PlayerGrid[i] <= 8) {
		return
	}
	c := int(s.PlayerGrid[i])
	js := make([]int, 0, 8-c)
	m := 0
	for dx := -1; dx <= +1; dx++ {
		for dy := -1; dy <= +1; dy++ {
			if 0 <= x+dx && x+dx < s.Width &&
				0 <= y+dy && y+dy < s.Height &&
				(dx != 0 || dy != 0) {
				j := (y+dy)*s.Width + (x + dx)
				if s.PlayerGrid[j] == Flagged {
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
			s.OpenCell(jx, jy)
			if s.Dead || s.Won {
				return
			}
		}
	}
}

func (s *GameState) Forfeit() {
	if !(s.Dead || s.Won) {
		s.Dead = true
	}
	s.RevealPlayerGrid()
}

func (s *GameState) RevealPlayerGrid() {
	if !(s.Dead || s.Won) {
		s.Dead = true
	}
	for i := range s.Grid {
		if s.PlayerGrid[i] == Flagged {
			if s.Grid[i] {
				s.PlayerGrid[i] = CorrectlyFlagged
			} else {
				s.PlayerGrid[i] = FalselyFlagged
			}
		} else if s.PlayerGrid[i] == Unknown || s.PlayerGrid[i] == Question {
			if s.Grid[i] {
				s.PlayerGrid[i] = UnflaggedMine
			} else {
				c := 0
				x := i % s.Width
				y := i / s.Width
				for dx := -1; dx <= +1; dx++ {
					for dy := -1; dy <= +1; dy++ {
						j := (y+dy)*s.Width + (x + dx)
						if 0 <= j && j < len(s.Grid) &&
							j != i && s.Grid[j] {
							c++
						}
					}
				}
				s.PlayerGrid[i] = CellState(c)
			}
		}
	}
}
