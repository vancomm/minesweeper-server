package mines

import (
	"fmt"
	"strings"
)

type mineCtx struct {
	grid             []bool
	width, height    int
	sx, sy           int
	allowBigPerturbs bool
}

func (ctx mineCtx) MineAt(x, y int) bool {
	return ctx.grid[y*ctx.width+x]
}

func (ctx mineCtx) Mines() (count int) {
	for _, s := range ctx.grid {
		if s {
			count++
		}
	}
	return
}

func (ctx mineCtx) Open(x, y int) CellState {
	if ctx.MineAt(x, y) {
		return Flagged /* *bang* */
	}
	n := 0
	for i := -1; i <= 1; i++ {
		if x+i < 0 || x+i >= ctx.width {
			continue
		}
		for j := -1; j <= 1; j++ {
			if y+j < 0 || y+j >= ctx.height {
				continue
			}
			if i == 0 && j == 0 {
				continue
			}
			if ctx.MineAt(x+i, y+j) {
				n++
			}
		}
	}
	return CellState(n)
}

func (ctx mineCtx) PrintGrid() string {
	var b strings.Builder
	for y := range ctx.height {
		for x := range ctx.width {
			var ch string
			if x == ctx.sx && y == ctx.sy {
				ch = "S "
			} else if ctx.grid[y*ctx.width+x] {
				ch = "* "
			} else {
				ch = "- "
			}
			fmt.Fprint(&b, ch)
		}
		fmt.Fprint(&b, "\n")
	}
	return b.String()
}

func (ctx *mineCtx) String() string {
	return fmt.Sprintf("%dx%d(%d:%d)", ctx.width, ctx.height, ctx.sx, ctx.sy)
}
