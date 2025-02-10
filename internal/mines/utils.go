package mines

type celltodo struct {
	next       []int
	head, tail int
}

func (std *celltodo) add(i int) {
	if std.tail >= 0 {
		std.next[std.tail] = i
	} else {
		std.head = i
	}
	std.tail = i
	std.next[i] = -1
}
