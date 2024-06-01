package game

type Cell struct {
	Flagged   bool
	opened    bool
	mineCount int
	mined     bool
}

func (c *Cell) Open() (mineCount int, mined bool) {
	c.opened = true
	return c.mineCount, c.mined
}

func (c Cell) Opened() bool {
	return c.opened
}

type Board []Cell
