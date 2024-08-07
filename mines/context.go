package mines

import (
	"fmt"
	"strconv"
	"strings"
)

type squaretodo struct {
	next       []int
	head, tail int
}

func (std *squaretodo) add(i int) {
	if std.tail >= 0 {
		std.next[std.tail] = i
	} else {
		std.head = i
	}
	std.tail = i
	std.next[i] = -1
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

func (s squareInfo) String() string {
	switch s {
	case Question:
		return "?"
	case Unknown:
		return "-"
	case Mine:
		return "*"
	case 0, 1, 2, 3, 4, 5, 6, 7, 8:
		return strconv.Itoa(int(s))
	default:
		return " "
	}
}

type gridInfo []squareInfo

func (g gridInfo) ToString(width int) string {
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

func (grid *gridInfo) knownSquares(
	w int,
	std *squaretodo,
	ctx *mineCtx,
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
						(*grid)[i] = Mine /* and don't open it! */
					} else {
						(*grid)[i] = ctx.Open(x+xx, y+yy)

						if (*grid)[i] == Mine { // assert grid[i] != -1
							Log.Fatal("bang") /* *bang* */
						}
					}
					std.add(i)
				}
			}
			bit <<= 1
		}
	}
}

type mineCtx struct {
	grid             []bool
	width, height    int
	sx, sy           int
	allowBigPerturbs bool
}

func (ctx mineCtx) MineAt(x, y int) bool {
	return ctx.grid[y*ctx.width+x]
}

func (ctx *mineCtx) Mines() (count int) {
	for _, s := range ctx.grid {
		if s {
			count++
		}
	}
	return
}

func (ctx *mineCtx) Open(x, y int) squareInfo {
	if ctx.MineAt(x, y) {
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
			if ctx.MineAt(x+i, y+j) {
				n++
			}
		}
	}
	return squareInfo(n)
}

func (ctx *mineCtx) PrintGrid() string {
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
