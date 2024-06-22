// source: https://git.tartarus.org/simon/puzzles.git/mines.c

package mines

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func iif[T any](c bool, t T, f T) T {
	if c {
		return t
	} else {
		return f
	}
}

func Main() {
	var (
		width  = 2
		height = 2
		mines  = 0
		grid   = []squareInfo{Unknown, Unknown, Unknown, Unknown}
		ctx    = &minectx{
			grid: []bool{false, false, false, false},
			w:    width, h: height,
			sx: 0, sy: 0,
			allowBigPerturbs: true,
		}
		res = MineSolve(
			width, height, mines, grid, mineOpen, minePerturb, ctx,
		)
	)

	switch res {
	case Stalled:
		fmt.Print("solution stalled")
	case Success:
		fmt.Print("solution succeeded")
	}
}
