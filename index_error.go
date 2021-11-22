package dolo

type IndexError struct {
	index int
	err   error
}

func NewIndexError(index int, err error) *IndexError {
	return &IndexError{index, err}
}
