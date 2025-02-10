package mines

type AssertionError struct {
	message string
}

// [AssertionError] implements [error]
func (e AssertionError) Error() string {
	return e.message
}
