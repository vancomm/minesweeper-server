package solver

import (
	"github.com/gammazero/deque"
	"github.com/sirupsen/logrus"
	"github.com/vancomm/minesweeper-server/game"
)

var log *logrus.Logger

func init() {
	log = logrus.New()
}

type Solver struct {
	*game.Game
	flagged      set[int]
	mineCounts   map[int]int
	inspectQueue deque.Deque[int]
}

func (s *Solver) flag(index int) {
	s.Game.SetFlagged(index, true)
	s.flagged[index] = void{}
	for _, i := range s.Game.GetExtendedOpenedNeighbors(index) {
		s.inspectQueue.PushBack(i)
	}
	s.inspectQueue.PushBack(index)
}

func (s *Solver) open(index int) (game.BoardUpdate, game.GameStatus) {
	return s.Game.Open(index)
}

func (s Solver) countRemainingMines(index int) (count int, ok bool) {
	count, ok = s.mineCounts[index]
	if !ok {
		return
	}
	var fromRow, toRow, fromCol, toCol = s.GetNeighborCoordsRange(index, 1)
	for r := fromRow; r <= toRow; r++ {
		for c := fromCol; c <= toCol; c++ {
			var (
				i     = s.CoordsToIndex(r, c)
				_, ok = s.flagged[i]
			)
			if i != index && ok {
				count++
			}
		}
	}
	return
}

// May panic if called with unopened cell index
func (s *Solver) inspectCell(index int) (update game.BoardUpdate, status game.GameStatus) {
	var untouchedNeighbors = s.Game.GetUntouchedNeighbors(index)
	if len(untouchedNeighbors) == 0 {
		return
	}

	var remainingMines, ok = s.countRemainingMines(index)
	if !ok {
		log.Fatal("tried to inspect an unopened cell")
	}
	if remainingMines == 0 {
		for _, i := range untouchedNeighbors {
			var upd, st = s.open(i)
			switch st {
			case game.Lost:
				log.Fatal("opened a mined cell while inspecting ")
			case game.Won:
				status = st
			}
			update = append(update, upd...)
		}
	}

	if remainingMines == len(untouchedNeighbors) {
		for _, i := range untouchedNeighbors {
			s.flag(i)
		}
		return
	}

	// find all opened cells in 5x5 radius
	var extendedOpenedNeighbors = s.GetExtendedOpenedNeighbors(index)
	for _, i := range extendedOpenedNeighbors {
		// find all unopened and unflagged neighbors of each such cell
		var (
			n      = s.GetUntouchedNeighbors(i)
			shared = Intersect(untouchedNeighbors, n)
		)
		if len(shared) == len(untouchedNeighbors) {
			continue
		}
		var rm, ok = s.countRemainingMines(i)
		if !ok {
			log.Fatal("tried to count remaining mines of unopened cell")
		}
		if len(untouchedNeighbors)-len(shared) == remainingMines-rm {
			for _, j := range untouchedNeighbors {
				s.flag(j)
			}
			return
		}
		if len(shared) == len(n) && remainingMines == rm {
			for _, i := range Complement(untouchedNeighbors, shared) {
				upd, st := s.open(i)
				switch st {
				case game.Lost:
					log.Fatal("opened a mined cell while inspecting")
				case game.Won:
					status = st
				}
				update = append(update, upd...)
			}
			return
		}
	}
	return
}

func (s *Solver) processInspectQueue() (status game.GameStatus) {
	for s.inspectQueue.Len() != 0 {
		var (
			index      = s.inspectQueue.PopFront()
			update, st = s.inspectCell(index)
		)
		if st != game.On {
			status = st
			return
		}
		s.processBoardUpdate(update)
	}
	return
}

func (s *Solver) processBoardUpdate(update game.BoardUpdate) {
	for _, c := range update {
		s.inspectQueue.PushBack(c.Index)
		s.mineCounts[c.Index] = c.MineCount
		for _, index := range s.Game.GetExtendedOpenedNeighbors(c.Index) {
			s.inspectQueue.PushBack(index)
		}
	}
}

func (s *Solver) Solve(firstUpdate game.BoardUpdate) (solved bool) {
	s.processBoardUpdate(firstUpdate)

	for updatedBoard := true; updatedBoard; {
		var status = s.processInspectQueue()
		if status != game.On {
			solved = status == game.Won
			return
		}
	}

	return
}

type Guesser struct {
	s                    *Solver
	interestingCells     []int
	allBorderCells       []int
	remainingBorderCells []int
	unreachableCells     []int
	minesRemaining       int
}

func (g *Guesser) Guess() (combinations []int) {
	var (
		interestingCellsCopy = Copy(g.interestingCells)
		originIndex          = -1
	)
	for _, possiblyInteresting := range interestingCellsCopy {
		if len(Intersect(g.s.GetNeighbors(possiblyInteresting), g.remainingBorderCells)) == 0 {
			var remainingMines, ok = g.s.countRemainingMines(possiblyInteresting)
			if !ok {
				log.Fatal("tried to count mines of closed cell")
			}
			if remainingMines == 0 {
				continue
			}
		}
	}
	return
}
