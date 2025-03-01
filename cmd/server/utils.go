package main

import (
	"hash/maphash"
	"math/rand/v2"
)

func createRand() *rand.Rand {
	return rand.New(rand.NewPCG(
		new(maphash.Hash).Sum64(), new(maphash.Hash).Sum64(),
	))
}
