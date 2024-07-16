package tree234_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vancomm/minesweeper-server/tree234"
)

type Item struct {
	Value int
}

func cmp(a, b *Item) int {
	if a.Value < b.Value {
		return -1
	}
	if a.Value > b.Value {
		return 1
	}
	return 0
}

func TestAdd(t *testing.T) {
	tree := tree234.New(cmp)
	for i := 1; i < 10; i++ {
		tree.Add(&Item{i})
	}

	assert.Equal(t, 9, tree.Count())
}

func TestIndex(t *testing.T) {
	var (
		empty *Item
		items []*Item
		tree  = tree234.New(cmp)
	)
	for i := 1; i < 10; i++ {
		item := &Item{i}
		items = append(items, item)
		tree.Add(item)
	}

	for i := range 15 {
		if i < len(items) {
			assert.Equal(t, items[i], tree.Index(i))
		} else {
			assert.Equal(t, empty, tree.Index(i))
		}
	}
}

func TestFindRelPos(t *testing.T) {
	var (
		items []*Item
		tree  = tree234.New(cmp)
	)
	for i := 1; i < 10; i++ {
		item := &Item{i}
		items = append(items, item)
		tree.Add(item)
	}

	_, index := tree.FindRelPos(items[1], tree234.Eq)
	assert.Equal(t, 1, index)

	_, index = tree.FindRelPos(items[7], tree234.Eq)
	assert.Equal(t, 7, index)
}

func TestDelete(t *testing.T) {
	var (
		empty *Item
		items []*Item
		tree  = tree234.New(cmp)
	)
	for i := 1; i < 10; i++ {
		item := &Item{i}
		items = append(items, item)
		tree.Add(item)
	}

	assert.Equal(t, empty, tree.Delete(&Item{10}))
	assert.Equal(t, items[7], tree.Delete(&Item{8}))
}
