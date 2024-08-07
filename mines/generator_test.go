package mines_test

import (
	"math/rand/v2"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/vancomm/minesweeper-server/mines"
	"github.com/vancomm/minesweeper-server/tree234"
)

func TestMain(m *testing.M) {
	// mines.Log.SetLevel(logrus.DebugLevel)
	mines.Log.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	// tree234.Log.SetLevel(logrus.DebugLevel)
	tree234.Log.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	m.Run()
}

func TestGen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		params mines.GameParams
	}{
		{
			name:   "9x9(10)",
			params: mines.GameParams{Width: 9, Height: 9, MineCount: 10, Unique: true},
		},
		{
			name:   "9x9(35)",
			params: mines.GameParams{Width: 9, Height: 9, MineCount: 35, Unique: true},
		},
		{
			name:   "16x16(40)",
			params: mines.GameParams{Width: 16, Height: 16, MineCount: 40, Unique: true},
		},
		{
			name:   "16x16(99)",
			params: mines.GameParams{Width: 16, Height: 16, MineCount: 99, Unique: true},
		},
		{
			name:   "30x16(99)",
			params: mines.GameParams{Width: 30, Height: 16, MineCount: 99, Unique: true},
		},
		{
			name:   "30x16(170)",
			params: mines.GameParams{Width: 30, Height: 16, MineCount: 170, Unique: true},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			params := test.params
			r := rand.New(rand.NewPCG(1, 2))
			for sx := 0; sx < params.Width; sx++ {
				for sy := 0; sy < params.Height; sy++ {
					t.Logf("%s @ %d:%d", test.name, sx, sy)
					_, err := mines.MineGen(params, sx, sy, r)
					if err != nil {
						t.Log(err)
						t.Errorf("could not generate game field %s", test.name)
					}
				}
			}
		})
	}
}
