package tree234

import "slices"

func iif[T any](condition bool, valueIfTrue T, valueIfFalse T) T {
	if condition {
		return valueIfTrue
	} else {
		return valueIfFalse
	}
}

func remove[T any](s []T, i int) []T {
	return append(s[:i], s[i+1:]...)
}

func insert[T any](s []T, i int, v T) []T {
	if i < len(s) {
		return slices.Insert(s, i, v)
	} else {
		return append(s, v)
	}
}
