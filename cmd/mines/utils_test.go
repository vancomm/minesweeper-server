package main

import "testing"

func TestByPiece(t *testing.T) {
	testCases := []struct {
		input string
		sep   string
		array []string
	}{
		{"a b c", " ", []string{"a", "b", "c"}},
		{"foo\nbar\nbaz\n\nbazz", "\n", []string{"foo", "bar", "baz", "", "bazz"}},
	}
	for _, test := range testCases {
		for i, p := range byPiece(test.input, test.sep) {
			if i < 0 || i >= len(test.array) {
				t.Errorf("byPiece returned an invalid index: %d", i)
			}
			if p != test.array[i] {
				t.Errorf("byPiece returned an incorrect piece: have %s, want %s",
					p, test.array[i])
			}
		}
	}
}
