package main

import (
	"fmt"
	"strings"
)

//go:generate stringer -type=GameMove
type GameMove uint8

const (
	Open GameMove = iota + 1
	Flag
	Chord
	LAST_MOVE
)

var ErrBadMove error

func init() {
	var allowedMoves []string
	for i := 1; i < int(LAST_MOVE); i++ {
		allowedMoves = append(allowedMoves, "'"+GameMove(i).String()+"'")
	}
	ErrBadMove = fmt.Errorf(
		"move must be one of %s",
		strings.ToLower(strings.Join(allowedMoves, ", ")),
	)
}

func decodeGameMove(s string) (move GameMove, err error) {
	switch strings.ToLower(s) {
	case "open":
		move = Open
	case "flag":
		move = Flag
	case "chord":
		move = Chord
	default:
		err = ErrBadMove
	}
	return
}
