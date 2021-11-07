package dolo

import (
	"github.com/boggydigital/nod"
	"io"
	"time"
)

const (
	timeoutSeconds = 10
	blockSize      = 32 * 1024
)

func CopyWithProgress(dst io.Writer, src io.ReadCloser, tpw nod.TotalProgressWriter) error {
	timer := time.AfterFunc(timeoutSeconds*time.Second, func() {
		src.Close()
	})

	for {
		timer.Reset(timeoutSeconds * time.Second)
		var reader io.Reader = src
		if tpw != nil {
			reader = io.TeeReader(src, tpw)
		}
		_, err := io.CopyN(dst, reader, blockSize)
		if err == io.EOF {
			break
		} else if err != nil {
			if tpw != nil {
				tpw.Error(err)
			}
			return err
		}
	}
	return nil
}
