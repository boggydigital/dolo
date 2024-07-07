package dolo

import "io"

type IndexSetter interface {
	Set(int, io.ReadCloser, chan *IndexResult, chan *IndexError)
	Get(int) (io.ReadCloser, error)
	Exists(int) bool
	Len() int
	IsUpdatedAfter(index int, since int64) (bool, error)
	ModTime(index int) (int64, error)
}
