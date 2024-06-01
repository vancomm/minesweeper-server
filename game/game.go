package game

import "math/rand"

type Game struct {
	Rows      int `schema:"rows,required"`
	Cols      int `schema:"cols,required"`
	Mines     int `schema:"mines,required"`
	FirstMove int `schema:"firstMove,required"`
	board     Board
	opened    int
}

type CellUpdate struct {
	Index     int
	MineCount int
	Mined     bool
}

type BoardUpdate []CellUpdate

type GameStatus int

const (
	On GameStatus = iota
	Won
	Lost
)

func (g Game) BoardLength() int {
	return len(g.board)
}

func (g Game) onlyMinesLeft() bool {
	return len(g.board)-g.opened == g.Mines
}

func (g *Game) Open(index int) (boardUpdate BoardUpdate, status GameStatus) {
	var (
		mineCount, mined = g.board[index].Open()
		cellUpdate       = CellUpdate{Index: index, MineCount: mineCount, Mined: mined}
	)

	g.opened++
	boardUpdate = append(boardUpdate, cellUpdate)

	if mined {
		status = Lost
		for i := range g.board {
			if !g.board[i].opened {
				var upd, _ = g.Open(i)
				boardUpdate = append(boardUpdate, upd...)
			}
		}
	} else if mineCount == 0 {
		for _, i := range g.GetNeighbors(index) {
			if !g.board[i].opened {
				var upd, _ = g.Open(i)
				boardUpdate = append(boardUpdate, upd...)
			}
		}
	}

	if status != Lost && g.onlyMinesLeft() {
		status = Won
	}
	return
}

func (g *Game) SetFlagged(index int, value bool) {
	g.board[index].Flagged = value
}

func (g Game) GetNeighborCoordsRange(index int, dist int) (fromRow, toRow, fromCol, toCol int) {
	var x, y = index / g.Rows, index % g.Cols
	fromRow, toRow = max(0, x-dist), min(x+dist, g.Rows-1)
	fromCol, toCol = max(0, y-dist), min(y+dist, g.Cols-1)
	return
}

func (g Game) CoordsToIndex(row int, col int) int {
	return row * g.Cols + col
}

func (g Game) IndexToCoords(index int) (row int, col int) {
	row, col = index / g.Cols, index % g.Cols
	return
}

func (g *Game) Chord(index int) (update BoardUpdate, status GameStatus) {
	if !g.board[index].opened {
		return
	}
	var (
		fromRow, toRow, fromCol, toCol = g.GetNeighborCoordsRange(index, 1)
		flagCount                      = 0
	)
	for r := fromRow; r <= toRow; r++ {
		for c := fromCol; c <= toCol; c++ {
			var i = g.CoordsToIndex(r, c)
			if g.board[i].Flagged {
				flagCount++
			}
		}
	}
	if g.board[index].mineCount != flagCount {
		return
	}
	for r := fromRow; r <= toRow; r++ {
		for c := fromCol; c <= toCol; c++ {
			var i = r*g.Cols + c
			if i != index && !g.board[i].Flagged && !g.board[i].opened {
				upd, st := g.Open(i)
				if st == Lost || (st == Won && status != Lost) {
					status = st
				}
				update = append(update, upd...)
			}
		}
	}
	if status != Lost && g.onlyMinesLeft() {
		status = Won
	}
	return
}

func (g *Game) getNeighbors(index int, dist int) (indices []int) {
	var (
		fromRow, toRow, fromCol, toCol = g.GetNeighborCoordsRange(index, dist)
		rows, cols                     = toRow - fromRow, toCol - fromCol
		neighborCount                  = rows*cols - 1
		neighborIndex                  = 0
	)
	indices = make([]int, neighborCount)
	for r := fromRow; r <= toRow; r++ {
		for c := fromCol; c <= toCol; c++ {
			var i = r*g.Cols + c
			if i != index {
				indices[neighborIndex] = i
				neighborIndex++
			}
		}
	}
	return
}

func (g *Game) getOpenedNeighbors(index int, dist int) (indices []int) {
	var (
		fromRow, toRow, fromCol, toCol = g.GetNeighborCoordsRange(index, dist)
		rows, cols                     = toRow - fromCol, toCol - fromCol
		neighborCount                  = rows*cols - 1
		neighborIndex                  = 0
	)
	indices = make([]int, neighborCount)
	for r := fromRow; r <= toRow; r++ {
		for c := fromCol; c <= toCol; c++ {
			var i = r*g.Cols + c
			if i != index && g.board[i].Opened() {
				indices[neighborIndex] = i
				neighborIndex++
			}
		}
	}
	return
}

func (g *Game) getUntouchedNeighbors(index int, dist int) (indices []int) {
	var (
		fromRow, toRow, fromCol, toCol = g.GetNeighborCoordsRange(index, dist)
		rows, cols                     = toRow - fromCol, toCol - fromCol
		neighborCount                  = rows*cols - 1
		neighborIndex                  = 0
	)
	indices = make([]int, neighborCount)
	for r := fromRow; r <= toRow; r++ {
		for c := fromCol; c <= toCol; c++ {
			var i = r*g.Cols + c
			if i != index && !g.board[i].Opened() && !g.board[i].Flagged {
				indices[neighborIndex] = i
				neighborIndex++
			}
		}
	}
	return
}

func (g *Game) GetNeighbors(index int) (neighbors []int) {
	return g.getNeighbors(index, 1)
}

func (g *Game) GetExtendedOpenedNeighbors(index int) (neighbors []int) {
	return g.getOpenedNeighbors(index, 2)
}

func (g *Game) GetUntouchedNeighbors(index int) (neighbors []int) {
	return g.getUntouchedNeighbors(index, 1)
}

func (g *Game) ResetBoard() {
	for i := range g.board {
		g.board[i].Flagged = false
		g.board[i].opened = false
	}
	g.opened = 0
}

func (g *Game) Random() (firstUpdate BoardUpdate) {
	var boardLength = g.Rows * g.Cols
	g.board = make(Board, boardLength)
	for i := range boardLength {
		g.board[i] = Cell{}
	}
	for plantedMines := 0; plantedMines < g.Mines; {
		var index = rand.Intn(boardLength)
		if index != g.FirstMove && !g.board[index].mined {
			g.board[index].mined = true
			plantedMines++
			for _, i := range g.GetNeighbors(index) {
				g.board[i].mineCount++
			}
		}
	}
	firstUpdate, _ = g.Open(g.FirstMove)
	return
}
