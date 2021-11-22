package dolo

type IndexResult struct {
	index  int
	result bool
}

func NewIndexResult(index int, result bool) *IndexResult {
	return &IndexResult{index, result}
}
