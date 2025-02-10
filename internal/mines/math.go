package mines

func absDiff(x, y int) int {
	if x > y {
		return x - y
	}
	return y - x
}
