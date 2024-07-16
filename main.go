package main

import (
	"github.com/vancomm/minesweeper-server/mines"
)

func main() {
	mines.MineGen(mines.GameParams{Width: 10, Height: 10, MineCount: 10, Unique: true}, 0, 0)
}
