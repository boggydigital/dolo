package dolo

type progress struct {
	total   uint64
	current uint64
	notify  func(uint64, uint64)
}

func (pr *progress) Write(p []byte) (int, error) {
	n := len(p)
	pr.current += uint64(n)
	if pr.notify != nil {
		pr.notify(pr.current, pr.total)
	}
	return n, nil
}
