package main

import (
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"
)

type Cell struct {
	X                         int  `json:"x"`
	Y                         int  `json:"y"`
	Mined                     bool `json:"mined"`
	MineCount                 int  `json:"mine_count"`
	covered                   bool
	flagged                   bool
	flaggedCount              int
	neighbors                 []*Cell
	neighborsCoveredUnflagged []*Cell
}

func (c Cell) Equal(other Cell) bool {
	return c.X == other.X && c.Y == other.Y
}

func (c Cell) remainingMines() int {
	return c.MineCount - c.flaggedCount
}

func (c Cell) missingFlags() int {
	var missing = c.MineCount
	for _, neighbor := range c.neighbors {
		if neighbor.flagged {
			missing--
		}
	}
	return missing
}

func (c *Cell) findTouching(borderingCells []*Cell) []*Cell {
	return Intersect(c.neighbors, borderingCells)
}

func NewCell(x int, y int) *Cell {
	var cell = &Cell{
		X:       x,
		Y:       y,
		covered: true,
	}
	return cell
}

type Board struct {
	Height           int `schema:"height,required"`
	Width            int `schema:"width,required"`
	Mines            int `schema:"mines,required"`
	FirstMove        int `schema:"first_move,required"`
	cells            []*Cell
	interestingCells []*Cell
	flagCount        int
}

func (b Board) solved() bool {
	return b.Mines == b.flagCount
}

func (b Board) indexToCoords(index int) (int, int) {
	return index / b.Height, index % b.Height
}

func (b Board) coordsToIndex(x int, y int) int {
	return x*b.Width + y
}

func (b Board) indexOf(cell *Cell) int {
	return b.coordsToIndex(cell.X, cell.Y)
}

func (b Board) Cells() []*Cell {
	return b.cells
}

func (b *Board) resetCells() {
	var boardSize = b.Height * b.Width
	b.cells = make([]*Cell, boardSize)
	b.flagCount = 0
	for index := range boardSize {
		var (
			x, y = b.indexToCoords(index)
			cell = NewCell(x, y)
		)
		b.cells[index] = cell
		// log.WithFields(logrus.Fields{
		// 	"x": x,
		// 	"y": y,
		// }).Debug("created cell")
	}
	// log.WithField("cells", b.cells).Debug("finished creating cells")
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
					cell.neighbors[neighborNum] = b.cells[neighborIndex]
					neighborNum++
				}
			}
		}
		cell.neighborsCoveredUnflagged = Duplicate(cell.neighbors)
		// log.WithFields(logrus.Fields{
		// 	"cell":                      cell,
		// 	"neighbors":                 cell.neighbors,
		// 	"neighborsCoveredUnflagged": cell.neighborsCoveredUnflagged,
		// }).Debug("found neighbors")
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
		var index = rand.Intn(boardSize)
		if index == b.FirstMove {
			continue
		}
		var cell = b.cells[index]
		if cell.Mined {
			continue
		}
		cell.Mined = true
		for _, neighbor := range cell.neighbors {
			neighbor.MineCount++
		}
		plantedMines++
	}
}

func (b *Board) uncover(cell *Cell) {
	cell.covered = false
	for _, neighbor := range cell.neighbors {
		neighbor.neighborsCoveredUnflagged = FilterEqual(neighbor.neighborsCoveredUnflagged, cell)
	}
}

func (b *Board) flag(cell *Cell) {
	cell.flagged = true
	for _, neighbor := range cell.neighbors {
		neighbor.flaggedCount++
		neighbor.neighborsCoveredUnflagged = FilterEqual(neighbor.neighborsCoveredUnflagged, cell)
	}
	// for _, neighbor := range b.findExtendedUncoveredNeighbors(cell) {
	// 	if !Contains(b.interestingCells, neighbor) {
	// 		b.interestingCells = append(b.interestingCells, neighbor)
	// 	}
	// }
	b.flagCount++
}

func (b Board) findExtendedUncoveredNeighbors(cell *Cell) (extendedNeighbors []*Cell) {
	var (
		minRow, maxRow = max(0, cell.X-2), min(cell.X+2, b.Height-1)
		minCol, maxCol = max(0, cell.Y-2), min(cell.Y+2, b.Width-1)
		cellIndex      = b.indexOf(cell)
	)
	for x := minRow; x <= maxRow; x++ {
		for y := minCol; y <= maxCol; y++ {
			if index := b.coordsToIndex(x, y); index != cellIndex {
				if neighbor := b.cells[index]; !neighbor.covered {
					extendedNeighbors = append(extendedNeighbors, neighbor)
				}
			}
		}
	}
	return
}

func (b Board) findSharedCoveredUnflaggedNeighbors(first *Cell, second *Cell) []*Cell {
	return Intersect(first.neighborsCoveredUnflagged, second.neighborsCoveredUnflagged)
}

func (b *Board) updateCells() bool {
	for index, cell := range Duplicate(b.interestingCells) {
		cell.remainingMines()
		var (
			remainingMines        = cell.remainingMines()
			coveredUnflaggedCount = len(cell.neighborsCoveredUnflagged)
		)
		// check if interesting
		if coveredUnflaggedCount == 0 {
			RemoveAt(&b.interestingCells, index)
			return true
		}
		// remaining mines == 0
		if remainingMines == 0 {
			RemoveAt(&b.interestingCells, index)
			for _, neighbor := range Duplicate(cell.neighborsCoveredUnflagged) {
				b.uncover(neighbor)
			}
			return true
		}
		// remaining mines == unflagged squares
		if remainingMines == coveredUnflaggedCount {
			RemoveAt(&b.interestingCells, index)
			for _, neighbor := range Duplicate(cell.neighborsCoveredUnflagged) {
				b.flag(neighbor)
				for _, neighNeighbor := range b.findExtendedUncoveredNeighbors(neighbor) {
					if !Contains(b.interestingCells, neighNeighbor) {
						b.interestingCells = append(b.interestingCells, neighbor)
					}
				}
			}
			return true
		}
		// shared neighbors
		for _, neighbor := range b.findExtendedUncoveredNeighbors(cell) {
			var sharedNeighbors = b.findSharedCoveredUnflaggedNeighbors(cell, neighbor)
			// log.WithField("shared", shared_neighbors).Debug("found shared neighbors")
			if len(sharedNeighbors) < len(cell.neighborsCoveredUnflagged) {
				// flag all cells that are definitely mines
				if len(cell.neighborsCoveredUnflagged)-len(sharedNeighbors) == cell.remainingMines()-neighbor.remainingMines() {
					for _, n := range Duplicate(cell.neighborsCoveredUnflagged) {
						if !Contains(sharedNeighbors, n) {
							b.flag(n)
						}
					}
					return true
				}
				// uncover all cells that are definitely not mines
				if len(sharedNeighbors) == len(neighbor.neighborsCoveredUnflagged) {
					if neighbor.remainingMines() == cell.remainingMines() {
						for _, n := range Duplicate(cell.neighborsCoveredUnflagged) {
							if !Contains(sharedNeighbors, n) {
								b.uncover(n)
							}
						}
						return true
					}
				}
			}
		}
		// nothing can be done with the square
		RemoveAt(&b.interestingCells, index)
	}
	return false
}

func (b Board) findFlagCombinations(
	interestingCells []*Cell,
	allBorderingCells []*Cell,
	remainingBorderingCells []*Cell,
	unreachableCells []*Cell,
	remainingMines int,
) (combinations [][]int) {
	var (
		remainingInterestingCells = Duplicate(interestingCells)
		origin                    *Cell
	)
	for index, candidate := range Duplicate(interestingCells) {
		if len(candidate.findTouching(remainingBorderingCells)) == 0 {
			if candidate.missingFlags() == 0 {
				RemoveAt(&remainingInterestingCells, index)
			} else {
				return
			}
		} else {
			origin = candidate
			break
		}
	}
	if len(remainingInterestingCells) == 0 {
		if remainingMines > len(unreachableCells) {
			return
		}
		var combination []int
		for range len(allBorderingCells) {
			combination = append(combination, 0)
		}
		combinations = append(combinations, combination)
		return
	}
	var (
		missingFlags = origin.missingFlags()
		touching     = origin.findTouching(remainingBorderingCells)
	)
	if missingFlags > len(touching) || missingFlags > remainingMines {
		return
	}
	for _, neighbor := range touching {
		if !neighbor.flagged && Contains(remainingBorderingCells, neighbor) {
			neighbor.flagged = true // significance???
			var remainingBorderingCellsCopy = Filter(remainingBorderingCells, func(t *Cell) bool { return t.Equal(*neighbor) })
			if missingFlags == 1 {
				remainingInterestingCells = FilterEqual(remainingInterestingCells, neighbor)
				for _, n := range touching {
					remainingBorderingCellsCopy = FilterEqual(remainingBorderingCellsCopy, n)
				}
			}
			for _, possiblyFinished := range neighbor.neighbors {
				if !possiblyFinished.covered {
					var missingFlags2 = possiblyFinished.missingFlags() // FIXME naming
					if missingFlags2 < 0 {                              // WTF
						return
					}
					if missingFlags2 == 0 {
						for _, n := range possiblyFinished.neighbors {
							remainingBorderingCellsCopy = FilterEqual(remainingBorderingCellsCopy, n)
						}
					}
				}
			}
			var suffixes = b.findFlagCombinations(
				remainingInterestingCells,
				allBorderingCells,
				remainingBorderingCellsCopy,
				unreachableCells,
				remainingMines-1,
			)
			for _, suffix := range suffixes {
				var (
					combination   []int
					neighborIndex int
				)
				if index, ok := IndexOf(allBorderingCells, neighbor); ok {
					neighborIndex = index
				} else {
					neighborIndex = -1
				}
				for index := range allBorderingCells {
					if index == neighborIndex || suffix[index] == 1 {
						combination = append(combination, 1)
					} else {
						combination = append(combination, 0)
					}
				}
				combinations = append(combinations, combination)
			}
		}
	}
	return
}

func (b *Board) updateMineCount() bool {
	var (
		coveredCells     = Filter(b.cells, func(c *Cell) bool { return c.covered })
		borderingCells   []*Cell
		unreachableCells []*Cell
		interestingCells []*Cell
		remainingMines   = b.Mines - b.flagCount
	)
	for _, cell := range coveredCells {
		var borderFound = false
		for _, neighbor := range cell.neighbors {
			if !neighbor.covered {
				borderFound = true
				if !Contains(interestingCells, neighbor) {
					interestingCells = append(interestingCells, neighbor)
				}
			}
		}
		if borderFound {
			borderingCells = append(borderingCells, cell)
		} else {
			unreachableCells = append(unreachableCells, cell)
		}
	}
	var safeCells = Duplicate(borderingCells)
	if remainingMines > 0 && remainingMines < 5 && len(coveredCells) < 10 {
		var correctCombinations = b.findFlagCombinations(interestingCells, borderingCells, borderingCells, unreachableCells, remainingMines)
		if len(correctCombinations) == 0 {
			log.Warn("correctCombinations is empty")
		}
		for _, combination := range correctCombinations {
			safeCells = Filter(safeCells, func(cell *Cell) bool {
				if index, ok := IndexOf(borderingCells, cell); ok {
					return combination[index] == 1
				} else {
					return true
				}
			})
		}
		if len(correctCombinations) == 1 {
			var combination = correctCombinations[0]
			for index, cell := range borderingCells {
				if combination[index] == 0 {
					b.uncover(cell)
				}
			}
			return true
		} else if len(safeCells) > 0 {
			for _, cell := range safeCells {
				b.uncover(cell)
			}
			return true
		} else {
			for _, combination := range correctCombinations {
				if SumInt(combination...) != remainingMines {
					return false
				}
			}
			if len(unreachableCells) == 0 {
				return false
			}
			for _, cell := range unreachableCells {
				b.uncover(cell)
			}
			return true
		}
	}
	return false
}

func (b *Board) solve() bool {
	b.interestingCells = []*Cell{b.cells[b.FirstMove]}
	var updated = true
	for updated {
		log.WithFields(logrus.Fields{
			"interesting": b.interestingCells,
		}).Debug("about to update cells")
		updated = b.updateCells()
		if !updated {
			log.WithFields(logrus.Fields{
				"interesting": b.interestingCells,
			}).Debug("about to update mine count")
			updated = b.updateMineCount()
			time.Sleep(time.Second)
		}
	}
	return b.solved()
}

func (b *Board) Solvable() bool {
	var (
		solvable = false
		maxTries = 100
		tries    = 0
	)
	for !solvable || tries <= maxTries {
		log.WithFields(logrus.Fields{
			"tries":    tries,
			"maxTries": maxTries,
		}).Debug("about to start solving a board")
		b.init()
		b.randomizeMines()
		solvable = b.solve()
		tries++
		log.WithFields(logrus.Fields{
			"tries":    tries,
			"solvable": solvable,
			"board":    b,
		}).Debug("finished solving a board")
	}
	return solvable
}
