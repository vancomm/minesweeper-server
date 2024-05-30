package main

type Solver struct {
	height         int
	width          int
	firstMove      int
	mines          int
	cells          []Cell
	cellsToInspect []*Cell
	flagCount      int
}

func NewSolver(board Board) Solver {
	return Solver{
		height:    board.Rows,
		width:     board.Cols,
		firstMove: board.FirstMove,
		cells:     Duplicate(board.cells),
		mines:     board.Mines,
	}
}

func (s Solver) coordsToIndex(x, y int) int {
	return x*s.width + y
}

func (s Solver) at(x, y int) *Cell {
	return &s.cells[s.coordsToIndex(x, y)]
}

func (s *Solver) uncover(cell *Cell) {
	cell.covered = false
	for _, neighbor := range cell.neighbors {
		neighbor.untouchedNeighbors = Without(neighbor.untouchedNeighbors, cell)
	}
	// all uncovered cells that have common neighbors with uncovered one should be inspected
	for _, neighbor := range s.find2x2UncoveredNeighbors(cell) {
		if !Contains(s.cellsToInspect, neighbor) {
			s.cellsToInspect = append(s.cellsToInspect, neighbor)
		}
	}
}

func (s *Solver) flag(cell *Cell) {
	cell.flagged = true
	for _, neighbor := range cell.neighbors {
		neighbor.flaggedCount++
		neighbor.untouchedNeighbors = Without(neighbor.untouchedNeighbors, cell)
	}
	for _, neighbor := range s.find2x2UncoveredNeighbors(cell) {
		if !Contains(s.cellsToInspect, neighbor) {
			s.cellsToInspect = append(s.cellsToInspect, neighbor)
		}
	}
	s.flagCount++
}

// Find all uncovered cells within 2 blocks of `cell` as they have neighbor
// cells common with `cell`
//
//	. . . . . . .
//	. o o o o o .
//	. o o o o o .
//	. o o c o o .
//	. o o o o o .
//	. o o o o o .
//	. . . . . . .
func (s *Solver) find2x2UncoveredNeighbors(cell *Cell) (extendedNeighbors []*Cell) {
	var (
		minRow, maxRow = max(0, cell.X-2), min(cell.X+2, s.height-1)
		minCol, maxCol = max(0, cell.Y-2), min(cell.Y+2, s.width-1)
	)
	for x := minRow; x <= maxRow; x++ {
		for y := minCol; y <= maxCol && (x != cell.X || y != cell.Y); y++ {
			if neighbor := s.at(x, y); !neighbor.covered {
				extendedNeighbors = append(extendedNeighbors, neighbor)
			}
		}
	}
	return
}

func (s Solver) solved() bool {
	return s.flagCount == s.mines
}

func (s *Solver) Solve() bool {
	var firstCell = &s.cells[s.firstMove]

	s.cellsToInspect = append(s.cellsToInspect, firstCell)
	s.uncover(firstCell)

	for updated := true; updated; {
		updated = s.inspectCells()
		if !updated {
			updated = s.updateMineCount()
		}
	}

	return s.solved()
}

func (s *Solver) inspectCells() bool {
	for _, inspectedCell := range Duplicate(s.cellsToInspect) {
		defer func() {
			s.cellsToInspect = Without(s.cellsToInspect, inspectedCell)
		}()

		// check if interesting (???)
		var untouchedNeighborCount = len(inspectedCell.untouchedNeighbors)
		if untouchedNeighborCount == 0 {
			return true
		}

		// no more unflagged mines => uncover all untouched
		var cellRemainingMines = inspectedCell.remainingMines()
		if cellRemainingMines == 0 {
			var neighbors = Duplicate(inspectedCell.untouchedNeighbors)
			for _, neighbor := range neighbors {
				defer func(n *Cell) {
					s.uncover(n)
				}(neighbor)
			}
			return true
		}

		// remaining mines == unflagged squares => flag all untouched
		if cellRemainingMines == untouchedNeighborCount {
			for _, neighbor := range Duplicate(inspectedCell.untouchedNeighbors) {
				defer s.flag(neighbor)
			}
			return true
		}

		// check common neighbors
		for _, neighborWithCommonNeighbors := range s.find2x2UncoveredNeighbors(inspectedCell) {
			var commonNeighbors = Intersect(inspectedCell.untouchedNeighbors, neighborWithCommonNeighbors.untouchedNeighbors)
			if len(commonNeighbors) < len(inspectedCell.untouchedNeighbors) {
				// private cells equal private mines => flag all such cells
				var (
					inspectedCellPrivateSpace = len(inspectedCell.untouchedNeighbors) - len(commonNeighbors)
					remainingMinesDiff        = inspectedCell.remainingMines() - neighborWithCommonNeighbors.remainingMines()
				)
				if inspectedCellPrivateSpace == remainingMinesDiff {
					for _, n := range Duplicate(inspectedCell.untouchedNeighbors) {
						if !Contains(commonNeighbors, n) {
							defer s.flag(n)
						}
					}
					return true
				}

				// uncover all cells that are definitely not mines
				var (
					uncoveredCellHasNoPrivateSpace = len(commonNeighbors) == len(neighborWithCommonNeighbors.untouchedNeighbors)
					sameRemainingMines             = neighborWithCommonNeighbors.remainingMines() == inspectedCell.remainingMines()
				)
				if uncoveredCellHasNoPrivateSpace && sameRemainingMines {
					for _, n := range Duplicate(inspectedCell.untouchedNeighbors) {
						if !Contains(commonNeighbors, n) {
							defer s.uncover(n)
						}
					}
					return true
				}
			}
		}
	}
	return false
}

func (s *Solver) updateMineCount() bool {
	var (
		coveredCells     = Filter(s.cells, func(c Cell) bool { return c.covered })
		borderingCells   []*Cell
		unreachableCells []*Cell
		interestingCells []*Cell
		remainingMines   = s.mines - s.flagCount
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

	if !(remainingMines > 0 && remainingMines < 5 && len(coveredCells) < 10) {
		return false
	}

	var combinations = s.findFlagCombinations(
		interestingCells,
		borderingCells,
		borderingCells,
		unreachableCells,
		remainingMines,
	)
	if len(combinations) == 0 {
		log.Warn("combinations is empty")
	}

	var safeCells = Duplicate(borderingCells)
	for _, combination := range combinations {
		var nextSafeCells []*Cell
		for _, cell := range safeCells {
			if index, ok := IndexOf(borderingCells, cell); ok && combination[index] == 0 {
				nextSafeCells = append(nextSafeCells, cell)
			} else {
				nextSafeCells = append(nextSafeCells, cell)
			}
		}
		safeCells = nextSafeCells
	}

	if len(combinations) == 1 {
		var combination = combinations[0]
		for index, cell := range borderingCells {
			if combination[index] == 0 {
				s.uncover(cell)
			}
		}
		return true
	}

	if len(safeCells) > 0 {
		for _, cell := range safeCells {
			s.uncover(cell)
		}
		return true
	}

	for _, combination := range combinations {
		// all combinations must have the same number of mines
		if SumInt(combination...) != remainingMines {
			return false
		}
	}

	for _, cell := range unreachableCells {
		s.uncover(cell)
	}
	return len(unreachableCells) > 0
}

func (s *Solver) findFlagCombinations(
	candidates []*Cell,
	allBorderingCells []*Cell,
	remainingBorderingCells []*Cell,
	unreachableCells []*Cell,
	remainingMines int,
) (combinations [][]int) {
	var (
		remainingCandidates = Duplicate(candidates)
		origin              *Cell
	)
	for _, candidate := range Duplicate(candidates) {
		if len(Intersect(candidate.neighbors, remainingBorderingCells)) == 0 {
			if candidate.missingFlagCount() == 0 {
				remainingCandidates = Without(remainingCandidates, candidate)
			} else {
				return
			}
		} else {
			origin = candidate
			break
		}
	}
	if len(remainingCandidates) == 0 {
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
		missingFlags      = origin.missingFlagCount()
		neighborsOnBorder = Intersect(origin.neighbors, remainingBorderingCells)
	)
	if missingFlags > len(neighborsOnBorder) || missingFlags > remainingMines {
		return
	}
	for _, neighbor := range neighborsOnBorder {
		if !neighbor.flagged && Contains(remainingBorderingCells, neighbor) {
			neighbor.flagged = true // significance???
			var remainingBorderingCellsCopy = Without(remainingBorderingCells, neighbor)
			if missingFlags == 1 {
				remainingCandidates = Without(remainingCandidates, neighbor)
				for _, n := range neighborsOnBorder {
					remainingBorderingCellsCopy = Without(remainingBorderingCellsCopy, n)
				}
			}
			for _, possiblyFinished := range neighbor.neighbors {
				if !possiblyFinished.covered {
					var missingFlags2 = possiblyFinished.missingFlagCount() // FIXME naming
					if missingFlags2 < 0 {                                  // WTF
						return
					}
					if missingFlags2 == 0 {
						for _, n := range possiblyFinished.neighbors {
							remainingBorderingCellsCopy = Without(remainingBorderingCellsCopy, n)
						}
					}
				}
			}
			var suffixes = s.findFlagCombinations(
				remainingCandidates,
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
