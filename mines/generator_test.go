package mines_test

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/vancomm/minesweeper-server/mines"
	"github.com/vancomm/minesweeper-server/tree234"
)

func TestMain(m *testing.M) {
	mines.Log.SetLevel(logrus.DebugLevel)
	// tree234.Log.SetLevel(logrus.DebugLevel)
	tree234.Log.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})
	m.Run()
}

func TestGen(t *testing.T) {
	var (
		r      = rand.New(rand.NewPCG(1, 2))
		params = mines.GameParams{
			Width:     30,
			Height:    20,
			MineCount: 130,
			Unique:    true,
		}
		sx, sy   = 14, 7
		res, err = mines.MineGen(params, sx, sy, r)
	)

	require.Nil(t, err)

	var b strings.Builder
	fmt.Fprint(&b, "\n")
	for y := range params.Height {
		for x := range params.Width {
			var ch string
			if x == sx && y == sy {
				ch = "S "
			} else if res[y*params.Width+x] {
				ch = "* "
			} else {
				ch = "- "
			}
			fmt.Fprint(&b, ch)
		}
		fmt.Fprint(&b, "\n")
	}
	t.Log(b.String())

}
