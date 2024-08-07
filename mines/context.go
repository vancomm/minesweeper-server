package mines

import (
	"fmt"
	"strings"
)

type minectx struct {
	grid             []bool
	width, height    int
	sx, sy           int
	allowBigPerturbs bool
}

func (ctx *minectx) Mines() (count int) {
	for _, s := range ctx.grid {
		if s {
			count++
		}
	}
	return
}

func (ctx *minectx) Open(x, y int) squareInfo {
	if ctx.mineAt(x, y) {
		return Mine /* *bang* */
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
			if ctx.mineAt(x+i, y+j) {
				n++
			}
		}
	}
	return squareInfo(n)
}

func (ctx *minectx) String() string {
	return fmt.Sprintf("%dx%d(%d:%d)", ctx.width, ctx.height, ctx.sx, ctx.sy)
}

func (ctx *minectx) PrintGrid() string {
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

func (ctx minectx) mineAt(x, y int) bool {
	return ctx.grid[y*ctx.width+x]
}
