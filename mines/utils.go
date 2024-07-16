package mines

import "github.com/sirupsen/logrus"

var Log = logrus.New()

type word uint16

func (word word) bitCount() int {
	word = ((word & 0xAAAA) >> 1) + (word & 0x5555)
	word = ((word & 0xCCCC) >> 2) + (word & 0x3333)
	word = ((word & 0xF0F0) >> 4) + (word & 0x0F0F)
	word = ((word & 0xFF00) >> 8) + (word & 0x00FF)
	return int(word)
}

func iif[T any](condition bool, valueIfTrue, valueIfFalse T) T {
	if condition {
		return valueIfTrue
	} else {
		return valueIfFalse
	}
}

func repeat[T any](value T, times int) (res []T) {
	for range times {
		res = append(res, value)
	}
	return
}
