package mines

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func naiveBitCount(i int) (count int) {
	s := strconv.FormatInt(int64(i), 2)
	for _, char := range s {
		if char == '1' {
			count += 1
		}
	}
	return
}

func TestBitCount(t *testing.T) {
	for i := range 0xFFFF {
		require.Equal(t, naiveBitCount(i), word(i).bitCount())
	}
}
