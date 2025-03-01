package mines

import (
	"math/rand/v2"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/vancomm/minesweeper-server/internal/tree234"
)

func TestMain(m *testing.M) {
	tree234.Log.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	m.Run()
}

func TestSolvableGridGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	tests := []struct {
		name   string
		params GameParams
	}{
		{
			name:   "9x9(10)",
			params: GameParams{Width: 9, Height: 9, MineCount: 10, Unique: true},
		},
		{
			name:   "9x9(35)",
			params: GameParams{Width: 9, Height: 9, MineCount: 35, Unique: true},
		},
		{
			name:   "16x16(40)",
			params: GameParams{Width: 16, Height: 16, MineCount: 40, Unique: true},
		},
		{
			name:   "16x16(99)",
			params: GameParams{Width: 16, Height: 16, MineCount: 99, Unique: true},
		},
		{
			name:   "30x16(99)",
			params: GameParams{Width: 30, Height: 16, MineCount: 99, Unique: true},
		},
		{
			name:   "30x16(170)",
			params: GameParams{Width: 30, Height: 16, MineCount: 170, Unique: true},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			r := rand.New(rand.NewPCG(1, 2))
			for sx := range test.params.Width {
				for sy := range test.params.Height {
					_, err := test.params.newSolvableGrid(sx, sy, r)
					if err != nil {
						t.Log(err)
						t.Errorf("could not generate game %s @ %d:%d", test.name, sx, sy)
					}
				}
			}
		})
	}
}
