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

func TestWeirdCase(t *testing.T) {
	var (
		params        = mines.GameParams{Width: 16, Height: 16, MineCount: 99, Unique: true}
		sx            = 14
		sy            = 13
		hi     uint64 = 2189302816385926607
		lo     uint64 = 14000230605465082873
		r             = rand.New(rand.NewPCG(hi, lo))
	)
	_, err := mines.MineGen(params, sx, sy, r)
	if err != nil {
		t.Errorf("could not generate field")
	}
}

// func fieldToString(grid []bool, params mines.GameParams, sx, sy int) string {
// 	var b strings.Builder
// 	fmt.Fprint(&b, "\n")
// 	for y := range params.Height {
// 		for x := range params.Width {
// 			var ch string
// 			if x == sx && y == sy {
// 				ch = "S "
// 			} else if grid[y*params.Width+x] {
// 				ch = "* "
// 			} else {
// 				ch = "- "
// 			}
// 			fmt.Fprint(&b, ch)
// 		}
// 		fmt.Fprint(&b, "\n")
// 	}
// 	return b.String()
// }
