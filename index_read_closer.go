package dolo

import "io"

type indexReadCloser struct {
	index      int
	readCloser io.ReadCloser
}
