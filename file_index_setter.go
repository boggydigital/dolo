package dolo

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type fileIndexSetter struct {
	filenames []string
}

func NewFileIndexSetter(filenames []string) IndexSetter {
	return &fileIndexSetter{filenames: filenames}
}

func (fis *fileIndexSetter) Set(index int, src io.ReadCloser, results chan *IndexResult, errors chan *IndexError) {

	defer src.Close()

	if index < 0 || index >= len(fis.filenames) {
		errors <- NewIndexError(index, fmt.Errorf("file nextAvailable out of bounds"))
		return
	}

	dir, _ := filepath.Split(fis.filenames[index])
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, dirPerm); err != nil {
			errors <- NewIndexError(index, err)
		}
	}

	file, err := os.Create(fis.filenames[index])
	if err != nil {
		errors <- NewIndexError(index, err)
		return
	}

	// individual file set operations are not progressive
	if err := CopyWithProgress(file, src, nil); err != nil {
		errors <- NewIndexError(index, err)
		return
	}

	if err := file.Close(); err != nil {
		errors <- NewIndexError(index, err)
		return
	}

	results <- NewIndexResult(index, true)
}

func (fis *fileIndexSetter) Get(index int) (io.ReadCloser, error) {
	if index < 0 || index >= len(fis.filenames) {
		return nil, errors.New("file index out of bounds")
	}

	return os.Open(fis.filenames[index])
}

func (fis *fileIndexSetter) Len() int {
	return len(fis.filenames)
}

func (fis *fileIndexSetter) Exists(index int) bool {
	if index < 0 || index >= len(fis.filenames) {
		return false
	}

	if _, err := os.Stat(fis.filenames[index]); err == nil {
		return true
	}

	return false
}

func (fis *fileIndexSetter) IsUpdatedAfter(index int, since int64) (bool, error) {
	if index < 0 || index >= len(fis.filenames) {
		return false, nil
	}

	if stat, err := os.Stat(fis.filenames[index]); err == nil {
		return stat.ModTime().Unix() > since, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

func (fis *fileIndexSetter) ModTime(index int) (int64, error) {
	if index < 0 || index >= len(fis.filenames) {
		return -1, nil
	}

	if stat, err := os.Stat(fis.filenames[index]); err != nil {
		return -1, err
	} else {
		return stat.ModTime().Unix(), nil
	}
}
