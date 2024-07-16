package mines

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIif(t *testing.T) {
	assert.Equal(t, 1, iif(true, 1, 0))
	assert.Equal(t, 0, iif(false, 1, 0))
}

func TestRepeat(t *testing.T) {
	assert.Equal(t, []int(nil), repeat(1, 0))
	assert.Equal(t, []int{1, 1, 1}, repeat(1, 3))
	assert.Equal(t, []rune{'a', 'a', 'a', 'a', 'a'}, repeat('a', 5))
}

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
		assert.Equal(t, naiveBitCount(i), word(i).bitCount())
	}
}
