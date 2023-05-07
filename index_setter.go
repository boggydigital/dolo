package dolo

import "io"

type IndexSetter interface {
	Set(int, io.ReadCloser, chan *IndexResult, chan *IndexError)
	Get(int) (io.ReadCloser, error)
	Exists(int) bool
	Len() int
	IsModifiedAfter(index int, since int64) bool
}
