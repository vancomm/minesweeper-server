package mines

type GameParams struct {
	Width, Height, MineCount int
	Unique                   bool
}

type MineLayout struct {
	Mines []bool
	/*
	 * If we haven't yet actually generated the mine layout, here's
	 * all the data we will need to do so.
	 */
	MineCount int
	Unique    bool
}

type GameState struct {
	Width, Height, MineCount int
	Dead, Won, UsedSolve     bool
	Layout                   *MineLayout /* real mine positions */
	Grid                     []bool      /* player knowledge */
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

func New() *GameState {
	state := &GameState{}
	return state
}
