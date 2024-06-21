package main

import (
	"math/rand/v2"
	"reflect"
	"slices"

	"github.com/sirupsen/logrus"
)

type SquareInfo int8

const (
	Unknown SquareInfo = iota - 2
	Mine
	// 0-8 for empty with given number of mined neighbors
)

type minectx struct {
	grid             []bool
	w, h             int
	sx, sy           int
	allowBigPerturbs bool
}

func (ctx minectx) at(x, y int) bool {
	return ctx.grid[y*ctx.w+x]
}

type square struct {
	x, y, type_, random int
}

type perturbdelta int8

const (
	AssumeMine  perturbdelta = 1
	AssumeClear perturbdelta = -1
)

type perturbation struct {
	x, y  int
	delta perturbdelta
}

type perturbcb func(ctx *minectx, grid []SquareInfo, setx, sety int, mask int) []*perturbation

func bitcount16(inword int) int {
	word := (uint)(inword)
	word = ((word & 0xAAAA) >> 1) + (word & 0x5555)
	word = ((word & 0xCCCC) >> 2) + (word & 0x3333)
	word = ((word & 0xF0F0) >> 4) + (word & 0x0F0F)
	word = ((word & 0xFF00) >> 8) + (word & 0x00FF)
	return (int)(word)
}

func absDiff(x, y int) int {
	if x > y {
		return x - y
	}
	return y - x
}

func squarecmp(a, b *square) int {
	if a.type_ < b.type_ {
		return -1
	}
	if a.type_ > b.type_ {
		return 1
	}
	if a.random < b.random {
		return -1
	}
	if a.random > b.random {
		return 1
	}
	if a.y < b.y {
		return -1
	}
	if a.y > b.y {
		return 1
	}
	if a.x < b.x {
		return -1
	}
	if a.x > b.x {
		return 1
	}
	return 0
}

func minePerturb(ctx *minectx, grid []SquareInfo, setx, sety int, mask int) (ret []*perturbation) {
	if mask == 0 && !ctx.allowBigPerturbs {
		return
	}
	var sqlist []*square
	for range ctx.w * ctx.h {
		sqlist = append(sqlist, &square{})
	}
	n := 0
	for y := 0; y < ctx.h; y++ {
		for x := 0; x < ctx.w; x++ {
			if absDiff(y, ctx.sy) <= 1 && absDiff(x, ctx.sx) <= 1 {
				continue
			}
			if (mask == 0 && grid[y*ctx.w+x] == Unknown) ||
				(x >= setx && (x < setx+3) &&
					y >= sety && (y < sety+3) &&
					(mask&(1<<((y-sety)*3+(x-setx)))) != 0) {
				continue
			}

			sqlist[n].x = x
			sqlist[n].y = y

			if grid[y*ctx.w+x] != Unknown {
				sqlist[n].type_ = 3 // known square
			} else {
				/*
					Unknown square. Examine everything around it and see if it
					borders on any known squares. If it does, it's class 1,
					otherwise it's 2.
				*/
				sqlist[n].type_ = 2

				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if x+dx >= 0 && x+dx < ctx.w &&
							y+dy >= 0 && y+dy < ctx.h &&
							grid[(y+dy)*ctx.w+(x+dx)] != Unknown {
							sqlist[n].type_ = 1
							break
						}
					}
				}
			}
			sqlist[n].random = int(rand.Uint32())
			n++
		}
	}

	slices.SortFunc(sqlist, squarecmp)

	var nfull, nempty int
	if mask != 0 {
		for dy := 0; dy < 3; dy++ {
			for dx := 0; dx < 3; dx++ {
				if mask&(1<<(dy*3+dx)) != 0 {
					if setx+dx > ctx.w || sety+dy > ctx.h {
						log.WithFields(logrus.Fields{
							"dx": dx, "dy": dy,
						}).Fatal("out of range")
					}

					if ctx.grid[(sety+dy)*ctx.w+(setx+dx)] {
						nfull++
					} else {
						nempty++
					}
				}
			}
		}
	} else {
		for y := 0; y < ctx.h; y++ {
			for x := 0; x < ctx.w; x++ {
				if grid[y*ctx.w+x] == Unknown {
					nfull++
				} else {
					nempty++
				}
			}
		}
	}

	var (
		ntofill, ntoempty int
		tofill, toempty   []*square
	)
	for range Iif(mask != 0, 9, ctx.w*ctx.h) {
		tofill = append(tofill, &square{})
		toempty = append(toempty, &square{})
	}
	for i := 0; i < n; i++ {
		sq := sqlist[i]
		if ctx.grid[sq.y*ctx.w+sq.x] {
			toempty[ntoempty] = sq
			ntoempty++
		} else {
			tofill[ntofill] = sq
			ntofill++
		}
		if ntofill == nfull || ntoempty == nempty {
			break
		}
	}

	var setlist []int
	if ntofill != nfull && ntoempty != nempty {
		if ntoempty == 0 {
			log.Fatal("ntoempty is 0 when it must not be")
		}
		setlist = make([]int, ctx.w*ctx.h)
		i := 0
		if mask != 0 {
			for dy := 0; dy < 3; dy++ {
				for dx := 0; dx < 3; dx++ {
					if mask&(1<<(dy*3+dx)) != 0 {
						if setx+dx > ctx.w || sety+dy > ctx.h {
							log.WithFields(logrus.Fields{
								"dx": dx, "dy": dy,
							}).Fatal("out of range")
						}

						if !ctx.grid[(sety+dy)*ctx.w+(setx+dx)] {
							setlist[i] = (sety+dy)*ctx.w + (setx + dx)
							i++
						}
					}
				}
			}
		} else {
			for y := 0; y < ctx.h; y++ {
				for x := 0; x < ctx.w; x++ {
					if grid[y*ctx.w+x] == Unknown {
						if !ctx.grid[y*ctx.w+x] {
							setlist[i] = y*ctx.w + x
							i++
						}
					}
				}
			}
		}

		if i <= ntoempty {
			log.WithFields(logrus.Fields{
				"i": i, "ntoempty": ntoempty,
			}).Fatal("i must be less than ntoempty")
		}

		for k := 0; k < ntoempty; k++ {
			index := k + rand.IntN(i-k)
			setlist[k], setlist[index] = setlist[index], setlist[k]
		}
	} else {
		setlist = nil
	}

	var (
		todo        []*square
		ntodo       int
		dtodo, dset perturbdelta
	)
	if ntofill == nfull {
		todo = tofill
		ntodo = ntofill
		dtodo = AssumeMine
		dset = AssumeClear
		toempty = nil
	} else {
		todo = toempty
		ntodo = ntoempty
		dtodo = AssumeClear
		dset = AssumeMine
		tofill = nil
	}

	var i int
	ret = make([]*perturbation, 2*ntodo)
	for i, t := range todo {
		ret[i] = &perturbation{
			x:     t.x,
			y:     t.y,
			delta: dtodo,
		}
	}

	if setlist != nil {
		if !reflect.DeepEqual(todo, toempty) {
			log.WithFields(logrus.Fields{
				"todo": todo, "toempty": toempty,
			}).Fatal("todo must be ")
		}
		for j := range ntoempty {
			ret[i] = &perturbation{
				x:     setlist[j] % ctx.w,
				y:     setlist[j] / ctx.w,
				delta: dset,
			}
			i++
		}
	} else if mask != 0 {
		for dy := range 3 {
			for dx := range 3 {
				if mask&(1<<(dy*3+dx)) != 0 {
					currval := Iif(ctx.grid[(sety+dy)*ctx.w+(setx+dx)], AssumeMine, AssumeClear)
					if dset == -currval {
						ret[i] = &perturbation{
							x:     setx + dx,
							y:     sety + dy,
							delta: dset,
						}
						i++
					}
				}
			}
		}
	} else {
		for y := range ctx.h {
			for x := range ctx.w {
				if grid[y*ctx.w+x] == Unknown {
					currval := Iif(ctx.grid[y*ctx.w+x], AssumeMine, AssumeClear)
					if dset == -currval {
						ret[i] = &perturbation{
							x:     x,
							y:     y,
							delta: dset,
						}
						i++
					}
				}
			}
		}
	}

	if i != len(ret) { // assert
		log.WithFields(logrus.Fields{
			"i": i, "ret.n": len(ret),
		}).Fatal("some perturbations have not generated")
	}

	sqlist = nil
	todo = nil

	for _, p := range ret {
		var (
			x     = p.x
			y     = p.y
			delta = p.delta
		)

		if (delta < 0) == (!ctx.grid[y*ctx.w+x]) { // assert
			log.Fatal("trying to add an existing mine or remove an absent one")
		}

		ctx.grid[y*ctx.w+x] = (delta > 0)

		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if x+dx == 0 && x+dx < ctx.w &&
					y+dy >= 0 && y+dy < ctx.h &&
					grid[(y+dy)*ctx.w+(x+dx)] != Unknown {
					if dx == 0 && dy == 0 {
						if delta > 0 {
							grid[y*ctx.w+x] = Mine
						} else {
							var minecount SquareInfo
							for dy2 := -1; dy2 <= 1; dy2++ {
								for dx2 := -1; dx2 <= 1; dx2++ {
									if x+dx2 >= 0 && x+dx2 < ctx.w &&
										y+dy2 >= 0 && y+dy2 < ctx.h &&
										ctx.grid[(y+dy2)*ctx.w+(x+dx2)] {
										minecount++
									}
								}
							}
							grid[y*ctx.w+x] = minecount
						}
					} else {
						if grid[(y+dy)*ctx.w+(x+dx)] >= 0 {
							grid[(y+dy)*ctx.w+(x+dx)] += SquareInfo(delta)
						}
					}
				}
			}
		}
	}
	return
}

type set struct {
	x, y, mask, mines int
	todo              bool
	next, prev        *set
}

func setcmp(a, b *set) int {
	if a.y < b.y {
		return -1
	}
	if a.y > b.y {
		return 1
	}
	if a.x < b.x {
		return -1
	}
	if a.x > b.x {
		return 1
	}
	if a.mask < b.mask {
		return -1
	}
	if a.mask > b.mask {
		return 1
	}
	return 0
}

type setstore struct {
	sets                 *Tree234[set]
	todo_head, todo_tail *set
}

func NewSetStore() *setstore {
	return &setstore{
		sets: NewTree234(setcmp),
	}
}

func setMunge(x1, y1, mask1, x2, y2, mask2 int, diff bool) int {
	if absDiff(x2, x1) >= 3 || absDiff(y2, y1) >= 3 {
		mask2 = 0
	} else {
		for x2 > x1 {
			mask2 &= ^(4 | 32 | 256)
			mask2 <<= 1
			x2--
		}
		for x2 < x1 {
			mask2 &= ^(1 | 8 | 64)
			mask2 >>= 1
			x2++
		}
		for y2 > y1 {
			mask2 &= ^(64 | 128 | 256)
			mask2 <<= 3
			y2--
		}
		for y2 < y1 {
			mask2 &= ^(1 | 2 | 4)
			mask2 >>= 3
			y2++
		}
	}

	if diff {
		mask2 ^= 511
	}

	return mask1 & mask2
}

func (ss *setstore) addTodo(s *set) {
	if s.todo {
		return
	}

	s.prev = ss.todo_tail
	if s.prev != nil {
		s.prev.next = s
	} else {
		ss.todo_head = s
		ss.todo_tail = s
		s.next = nil
		s.todo = true
	}
}

func (ss *setstore) add(x, y, mask, mines int) {
	if mask == 0 { // assert mask != 0
		log.Fatal("mask is 0")
	}

	for mask&(1|8|64) == 0 {
		mask >>= 1
		x++
	}
	for mask&(1|2|4) == 0 {
		mask >>= 3
		y++
	}

	s := &set{
		x:     x,
		y:     y,
		mask:  mask,
		mines: mines,
		todo:  false,
	}

	if ss.sets.Add(s) != s {
		s = nil
		return
	}

	ss.addTodo(s)
}

func (ss *setstore) remove(s *set) {
	var (
		next = s.next
		prev = s.prev
	)

	if prev != nil {
		prev.next = next
	} else if s == ss.todo_head {
		ss.todo_head = next
	}

	if next != nil {
		next.prev = prev
	} else if s == ss.todo_tail {
		ss.todo_tail = prev
	}

	s.todo = false

	ss.sets.Del(s)
}

func (ss *setstore) overlap(x, y, mask int) (ret []*set) {
	var xx, yy int
	for xx = x - 3; xx < x+3; xx++ {
		for yy = y - 3; yy < y+3; yy++ {
			stmp := &set{
				x:    x,
				y:    y,
				mask: 0,
			}
			if el, pos := ss.sets.FindRelPos(stmp, Rel234Ge); el != nil {
				for s := ss.sets.Index(pos); s != nil &&
					s.x == xx && s.y == yy; {
					if setMunge(x, y, mask, s.x, s.y, s.mask, false) != 0 {
						ret = append(ret, s)
					}
					pos++
				}
			}
		}
	}
	// ret = append(ret, nil)
	return
}

func (ss *setstore) todo() (ret *set) {
	ret = ss.todo_head
	if ret != nil {
		ss.todo_head = ret.next
		if ss.todo_head != nil {
			ss.todo_head.prev = nil
		} else {
			ss.todo_tail = nil
		}
		ret.next, ret.prev = nil, nil
		ret.todo = false
	}
	return
}

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

type opencb func(*minectx, int, int) SquareInfo

func knownSquares(
	w, h int,
	std *squaretodo,
	grid []SquareInfo,
	open opencb, openctx *minectx,
	x, y, mask int, mine bool,
) {
	bit := 1
	for yy := range 3 {
		for xx := range 3 {
			if mask&bit != 0 {
				i := (y+yy)*w + (x + xx)
				if grid[i] == Unknown {
					if mine {
						grid[i] = Mine
					} else {
						grid[i] = open(openctx, x+xx, y+yy)

						if grid[i] == Mine { // assert grid[i] != -1
							log.Fatal("boom")
						}
					}
					std.add(i)
				}
			}
			bit <<= 1
		}
	}
}

/* x and y must be in range of ctx.Grid's w and h */
func mineOpen(ctx *minectx, x, y int) SquareInfo {
	if ctx.at(x, y) {
		return Mine
	}
	var n SquareInfo
	for i := -1; i <= 1; i++ {
		if x+i < 0 || x+i >= ctx.w {
			continue
		}
		for j := -1; j <= 1; j++ {
			if y+j < 0 || y+j >= ctx.h {
				continue
			}
			if i == 0 && j == 0 {
				continue
			}
			if ctx.at(x+i, y+j) {
				n++
			}
		}
	}
	return n
}

type SolveResult int8

const (
	Stalled SolveResult = iota - 1
	Success
	// values >0 mean given number of perturbations was required
)

func mineSolve(
	w, h, n int,
	grid []SquareInfo,
	open opencb,
	perturb perturbcb,
	ctx *minectx,
) (res SolveResult) {
	var (
		ss        = NewSetStore()
		std       = &squaretodo{}
		nperturbs = 0
	)
	std.next = make([]int, w*h)
	std.head, std.tail = -1, -1
	for y := range h {
		for x := range w {
			i := y*w + x
			if grid[i] != Unknown {
				std.add(i)
			}
		}
	}

	for {
		doneSomething := false

		for std.head != -1 {
			i := std.head
			std.head = std.next[i]
			if std.head == -1 {
				std.tail = -1
			}
			x, y := i%w, i/w
			if mines := grid[i]; mines >= 0 {
				bit := 1
				val := 0
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if x+dx < 0 || x+dx >= w || y+dy < 0 || y+dy >= h {
							// do nothing
						} else if grid[i+dy*w+dx] == Mine {
							mines--
						} else if grid[i+dy*w+dx] == Unknown {
							val |= bit
						}
						bit <<= 1
					}
				}
				if val != 0 {
					ss.add(x-1, y-1, val, int(mines))
				}
			}
			{
				for _, s := range ss.overlap(x, y, 1) {
					newmask := setMunge(s.x, s.y, s.mask, x, y, 1, true)
					newmines := s.mines - Iif(grid[i] == Mine, 1, 0)
					if newmask != 0 {
						ss.add(s.x, s.y, newmask, newmines)
					}
					ss.remove(s)
				}
			}
			doneSomething = true
		}

		if s := ss.todo(); s != nil {
			if s.mines == 0 || s.mines == bitcount16(s.mask) {
				knownSquares(w, h, std, grid, open, ctx, s.x, s.y, s.mask, s.mines != 0)
				continue
			}
			for _, s2 := range ss.overlap(s.x, s.y, s.mask) {
				swing := setMunge(s.x, s.y, s.mask, s2.x, s2.y, s2.mask, true)
				s2wing := setMunge(s2.x, s2.y, s2.mask, s.x, s.y, s.mask, true)
				swc := bitcount16(swing)
				s2wc := bitcount16(s2wing)

				if (swc == s.mines-s2.mines) || (s2wc == s2.mines-s.mines) {
					knownSquares(w, h, std, grid, open, ctx,
						s.x, s.y, swing,
						(swc == s.mines-s2.mines))
					knownSquares(w, h, std, grid, open, ctx,
						s2.x, s2.y, s2wing,
						(s2wc == s2.mines-s.mines))
					continue
				}

				if swc == 0 && s2wc != 0 {
					ss.add(s2.x, s2.y, s2wing, s2.mines-s.mines)
				} else if s2wc == 0 && swc != 0 {
					ss.add(s.x, s.y, swing, s.mines-s2.mines)
				}
			}
			doneSomething = true
		} else if n >= 0 {
			/*
				Global deduction
			*/

			squaresleft := 0
			minesleft := n
			for i := range w * h {
				if grid[i] == Mine {
					minesleft--
				} else if grid[i] == Unknown {
					squaresleft++
				}
			}

			if squaresleft == 0 {
				break
			}

			if minesleft == 0 || minesleft == squaresleft {
				for i := range w * h {
					if grid[i] == Unknown {
						knownSquares(w, h, std, grid, open, ctx,
							i%w, i/w, 1, minesleft != 0)
					}
				}
				continue
			}

			setused := make([]bool, 10)
			nsets := ss.sets.Count()

			if nsets <= len(setused) {
				var sets []*set
				for i := range nsets {
					sets = append(sets, ss.sets.Index(i))
				}
				cursor := 0
				for {
					if cursor < nsets {
						ok := true
						for i := range cursor {
							if setused[i] && setMunge(
								sets[cursor].x, sets[cursor].y, sets[cursor].mask,
								sets[i].x, sets[i].y, sets[i].mask, false,
							) != 0 {
								ok = false
								break
							}
						}
						if ok {
							minesleft -= sets[cursor].mines
							squaresleft -= bitcount16(sets[cursor].mask)
						}
						setused[cursor] = ok
						cursor++
					} else {
						if squaresleft > 0 && (minesleft == 0 || minesleft == squaresleft) {
							for i := range w * h {
								if grid[i] == Unknown {
									outside := true
									y := i / w
									x := i % w
									for j := range nsets {
										if setused[j] &&
											setMunge(
												sets[j].x, sets[j].y, sets[j].mask,
												x, y, 1, false,
											) != 0 {
											outside = false
											break
										}
									}
									if outside {
										knownSquares(
											w, h, std, grid,
											open, ctx, x, y, 1, minesleft != 0,
										)
									}
								}
							}
							doneSomething = true
							break
						}
						cursor--
						for cursor >= 0 && !setused[cursor] {
							cursor--
						}
						if cursor >= 0 {
							minesleft += sets[cursor].mines
							squaresleft += bitcount16(sets[cursor].mask)
							setused[cursor] = false
							cursor++
						} else {
							break
						}
					}
				}
			}
		}

		if doneSomething {
			continue
		}

		nperturbs++
		var ret []*perturbation
		if c := ss.sets.Count(); c == 0 {
			ret = perturb(ctx, grid, 0, 0, 0)
		} else {
			s := ss.sets.Index(rand.IntN(c))
			ret = perturb(ctx, grid, s.x, s.y, s.mask)
		}
		if len(ret) > 0 {
			for _, p := range ret {
				if p.delta < 0 && grid[p.y*w+p.x] != Unknown {
					std.add(p.y*w + p.x)
				}
			}
		}
	}
	return
}

func main() {
	mineSolve(10, 10, 10, []SquareInfo{}, mineOpen, minePerturb, &minectx{})
}
