package main

import (
	"math/rand"

	"github.com/sirupsen/logrus"
)

type Cell struct {
	X                  int  `json:"x"`
	Y                  int  `json:"y"`
	Mined              bool `json:"mined"`
	MineCount          int  `json:"mineCount"`
	covered            bool
	flagged            bool
	flaggedCount       int
	neighbors          []*Cell
	untouchedNeighbors []*Cell
}

func (c Cell) Equal(other Cell) bool {
	return c.X == other.X && c.Y == other.Y
}

func (c Cell) remainingMines() int {
	return c.MineCount - c.flaggedCount
}

func (c Cell) missingFlagCount() int {
	var missing = c.MineCount
	for _, neighbor := range c.neighbors {
		if neighbor.flagged {
			missing--
		}
	}
	return missing
}

func NewCell(x int, y int) Cell {
	return Cell{
		X:       x,
		Y:       y,
		covered: true,
	}
}

type Board struct {
	Height    int `schema:"height,required"`
	Width     int `schema:"width,required"`
	Mines     int `schema:"mines,required"`
	FirstMove int `schema:"firstMove,required"`
	cells     []Cell
	flagCount int
}

func (b Board) Won() bool {
	return b.Mines == b.flagCount
}

func (b Board) indexToCoords(index int) (int, int) {
	return index / b.Height, index % b.Height
}

func (b Board) coordsToIndex(x, y int) int {
	return x*b.Width + y
}

func (b Board) Cells() []Cell {
	return b.cells
}

func (b Board) MinedCells() []*Cell {
	return Filter(b.cells, func(c Cell) bool { return c.Mined })
}

func (b *Board) resetCells() {
	var boardSize = b.Height * b.Width
	b.cells = make([]Cell, boardSize)
	b.flagCount = 0
	for index := range boardSize {
		var (
			x, y = b.indexToCoords(index)
			cell = NewCell(x, y)
		)
		b.cells[index] = cell
	}
}

func (b *Board) connectNeighbors() {
	var boardSize = b.Height * b.Width
	for index := range boardSize {
		var (
			x, y             = b.indexToCoords(index)
			cell             = b.cells[index]
			prevRow, nextRow = max(0, x-1), min(x+1, b.Height-1)
			prevCol, nextCol = max(0, y-1), min(y+1, b.Width-1)
			rows, cols       = (nextRow - prevRow + 1), (nextCol - prevCol + 1)
			neighborCount    = rows*cols - 1
			neighborNum      = 0
		)
		cell.neighbors = make([]*Cell, neighborCount)
		for i := prevRow; i <= nextRow; i++ {
			for j := prevCol; j <= nextCol; j++ {
				if neighborIndex := b.coordsToIndex(i, j); neighborIndex != index {
					cell.neighbors[neighborNum] = &b.cells[neighborIndex]
					neighborNum++
				}
			}
		}
		cell.untouchedNeighbors = Duplicate(cell.neighbors)
	}
}

func (b *Board) init() {
	b.resetCells()
	b.connectNeighbors()
}

func (b *Board) randomizeMines() {
	var (
		boardSize    = b.Height * b.Width
		plantedMines = 0
	)
	for plantedMines < b.Mines {
		var (
			index = rand.Intn(boardSize)
			cell  = b.cells[index]
		)
		if index == b.FirstMove || cell.Mined {
			continue
		}
		cell.Mined = true
		for _, neighbor := range cell.neighbors {
			neighbor.MineCount++
		}
		plantedMines++
	}
}

func (b *Board) Uncover(cell *Cell) (updated []Cell, boom bool) {
	cell.covered = false
	updated = append(updated, *cell)

	if cell.Mined {
		boom = true
		for _, c := range b.cells {
			if c.covered {
				c.covered = false
				updated = append(updated, c)
			}
		}
		return
	}

	if cell.MineCount == 0 {
		for _, neighbor := range cell.neighbors {
			if neighbor.covered && !neighbor.Mined {
				var neighborUpdates, _ = b.Uncover(neighbor)
				updated = append(updated, neighborUpdates...)
			}
		}
	}

	return
}

func (b *Board) Solvable(maxTries int) (solved bool) {
	for tries := 0; tries <= maxTries && !solved; tries++ {
		log.WithFields(logrus.Fields{
			"board":    b,
			"mines":    b.MinedCells(),
			"maxTries": maxTries,
			"tries":    tries,
		}).Debug("starting solving attempt")
		b.init()
		b.randomizeMines()
		var solver = NewSolver(*b)
		solved = solver.Solve()
	}
	log.Debug("finished solving")
	return
}
