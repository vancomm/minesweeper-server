package main

import (
	"iter"
	"strings"
)

func byPiece(s string, sep string) iter.Seq2[int, string] {
	return func(yield func(int, string) bool) {
		i := 0
		found := true
		var piece string
		for found {
			piece, s, found = strings.Cut(s, sep)
			if !yield(i, piece) {
				return
			}
			i += 1
		}
	}
}
