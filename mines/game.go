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

func NewLayout(params GameParams, x, y int) *MineLayout {
	return &MineLayout{
		Mines:     MineGen(params, x, y),
		MineCount: params.MineCount,
		Unique:    params.Unique,
	}
}

type GameState struct {
	GameParams
	Dead, Won, UsedSolve bool
	Layout               MineLayout   /* real mine positions */
	Grid                 []squareInfo /* player knowledge */
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
	if s.Layout.Mines[i] {
		/*
		 * The player has landed on a mine. Bad luck. Expose the
		 * mine that killed them, but not the rest (in case they
		 * want to Undo and carry on playing).
		 */
		s.Dead = true
		s.Grid[i] = Exploded
		return -1
	}
	return 0
}

func New(params GameParams, x, y int) *GameState {
	return &GameState{
		GameParams: params,
		Layout:     *NewLayout(params, x, y),
	}
}
