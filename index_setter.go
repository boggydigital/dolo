package dolo

import "io"

type IndexSetter interface {
	Set(int, io.ReadCloser, chan *IndexResult, chan *IndexError)
	Exists(int) bool
	Len() int
}
