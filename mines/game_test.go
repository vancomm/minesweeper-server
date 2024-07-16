package mines

import (
	"testing"
)

func TestGame(t *testing.T) {
	var (
		width  = 2
		height = 2
		mines  = 0
		grid   = repeat(Unknown, 4)
		ctx    = &minectx{
			grid: repeat(false, 4),
			w:    width, h: height,
			sx: 0, sy: 0,
			allowBigPerturbs: true,
		}
		res = MineSolve(
			width, height, mines, grid, mineOpen, minePerturb, ctx,
		)
	)

	switch res {
	case Success:
		t.Log("solution succeeded")
	case Stalled:
		t.Error("solution stalled")
	}
}
