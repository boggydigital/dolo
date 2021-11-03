package dolo

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type fileIndexSetter struct {
	filenames []string
}

func NewFileIndexSetter(filenames []string) *fileIndexSetter {
	return &fileIndexSetter{filenames: filenames}
}

func (fis *fileIndexSetter) Set(index int, src io.ReadCloser, completion chan bool, errors chan error) {

	defer src.Close()

	if index < 0 || index >= len(fis.filenames) {
		errors <- fmt.Errorf("file current out of bounds")
		return
	}

	dir, _ := filepath.Split(fis.filenames[index])
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, dirPerm); err != nil {
			errors <- err
		}
	}

	file, err := os.Create(fis.filenames[index])
	if err != nil {
		errors <- err
		return
	}

	// individual file set operations are not progressive
	if err := Copy(file, src, nil); err != nil {
		errors <- err
		return
	}

	if err := file.Close(); err != nil {
		errors <- err
		return
	}

	completion <- true
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
