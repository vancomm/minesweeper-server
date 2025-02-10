package mines

import (
	"fmt"
	"strconv"
	"strings"
)

type CellState int8

const (
	Todo             CellState = -10
	Question         CellState = -3
	Unknown          CellState = -2
	Flagged          CellState = -1
	CorrectlyFlagged CellState = 64
	ExplodedMine     CellState = 65
	FalselyFlagged   CellState = 66
	UnflaggedMine    CellState = 67
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
)

func (s CellState) String() string {
	switch {
	case s == Question:
		return "?"
	case s == Unknown:
		return " "
	case s == Flagged:
		return "*"
	case 0 <= s && s <= 8:
		return strconv.Itoa(int(s))
	default:
		return "!"
	}
}

type Grid []CellState

func (g Grid) ToString(width int) string {
	var b strings.Builder
	for y := range len(g) / width {
		for x := range width {
			i := y*width + x
			if i >= len(g) {
				break
			}
			fmt.Fprint(&b, g[i].String()+" ")
		}
		fmt.Fprint(&b, "\n")

	}
	return b.String()
}

// panics [AssertionError]
func (grid *Grid) knownCells(
	w int, std *celltodo, ctx *mineCtx,
	x, y int, mask word, mine bool,
) {
	var bit word = 1
	for yy := range 3 {
		for xx := range 3 {
			if mask&bit != 0 {
				i := (y+yy)*w + (x + xx)

				/*
				 * It's possible that this square is _already_
				 * known, in which case we don't try to add it to
				 * the list twice.
				 */
				if (*grid)[i] == Unknown {
					if mine {
						(*grid)[i] = Flagged /* and don't open it! */
					} else {
						(*grid)[i] = ctx.Open(x+xx, y+yy)

						if (*grid)[i] == Flagged {
							panic(AssertionError{"grid[i] != -1"})
						}
					}
					std.add(i)
				}
			}
			bit <<= 1
		}
	}
}
