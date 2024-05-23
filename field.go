package main

type Cell struct {
	X          int `json:"x"`
	Y          int `json:"y"`
	neighbours []*Cell
}

func NewCell(x int, y int, neighbours []*Cell) *Cell {
	return &Cell{x, y, neighbours}
}

func CreateSolvableField(height int, width int, bombs int) [][]*Cell {
	field := make([][]*Cell, height)
	for i := range height {
		field[i] = make([]*Cell, width)
		for j := range width {
			var (
				numNeighbors = 8
				touchesTop  = i == 0
				bottomBorder  = i == (height - 1)
				westBorder   = j == 0
				eastBorder   = j == (width - 1)
			)
			topLeftCovered := touchesTop || 
			neighbors := make([]*Cell, numNeighbors)
			cell := NewCell(i, j, neighbors)
			field[i][j] = cell
		}
	}
	return field
}
