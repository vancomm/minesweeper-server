package main

import (
	"errors"
	"strconv"
	"strings"

	"github.com/vancomm/minesweeper-server/internal/mines"
)

// Maps known commands to number of arguments
var commandNargs = map[string]int{
	"g": 0,
	"o": 2,
	"f": 2,
	"c": 2,
	"r": 0,
}

func parseXY(twoStrings []string) (x int, y int, err error) {
	if x, err = strconv.Atoi(twoStrings[0]); err != nil {
		err = errors.New("first argument must be an int")
		return
	}
	if y, err = strconv.Atoi(twoStrings[1]); err != nil {
		err = errors.New("second argument must be an int")
		return
	}
	return
}

func executeCommand(g *mines.GameState, c string) (err error) {
	parts := strings.Split(c, " ")
	nargs, ok := commandNargs[parts[0]]
	if !ok {
		return errors.New("unknown command")
	}
	if nargs != len(parts)-1 {
		return errors.New("invalid number of arguments")
	}
	switch parts[0] {
	case "g":
		return
	case "o":
		if x, y, err := parseXY(parts[1:]); err != nil {
			return err
		} else if !g.ValidatePoint(x, y) {
			return errors.New("invalid square coordinates")
		} else {
			g.OpenCell(x, y)
		}
		return
	case "f":
		if x, y, err := parseXY(parts[1:]); err != nil {
			return err
		} else if !g.ValidatePoint(x, y) {
			return errors.New("invalid square coordinates")
		} else {
			g.FlagCell(x, y)
		}
		return
	case "c":
		if x, y, err := parseXY(parts[1:]); err != nil {
			return err
		} else if !g.ValidatePoint(x, y) {
			return errors.New("invalid square coordinates")
		} else {
			g.ChordCell(x, y)
		}
		return
	case "r":
		g.RevealMines()
		return
	}
	return errors.New("invalid command")
}
