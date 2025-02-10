// source: https://git.tartarus.org/simon/puzzles.git/mines.c

package mines

import (
	"fmt"
	"strings"
)

/* ----------------------------------------------------------------------
 * Grid generator which uses the [above] solver.
 */

type GameParams struct {
	Width, Height, MineCount int
	Unique                   bool
}

func (p GameParams) Unpack() (w int, h int, mc int, u bool) {
	return p.Width, p.Height, p.MineCount, p.Unique
}

func (p GameParams) Seed() string {
	u := 0
	if p.Unique {
		u = 1
	}
	return fmt.Sprintf("%d:%d:%d:%d", p.Width, p.Height, p.MineCount, u)
}

func ParseSeed(seed string) (*GameParams, error) {
	p := &GameParams{}
	u := 0
	sseed := strings.ReplaceAll(seed, ":", " ")
	n, err := fmt.Sscanf(
		sseed, "%d %d %d %d", &p.Width, &p.Height, &p.MineCount, &u,
	)
	if n != 4 || err != nil {
		return nil, fmt.Errorf(
			`invalid game params seed (sseed = "%s", n = %d, err = %w)`,
			sseed, n, err,
		)
	}
	p.Unique = u == 1
	return p, nil
}

func (p GameParams) PointInBounds(x, y int) bool {
	return y*p.Width+x < p.Width*p.Height
}
