package dolo

import (
	"github.com/boggydigital/nod"
	"io"
	"time"
)

const timeout = 10

func Copy(dst io.Writer, src io.ReadCloser, tpw nod.TotalProgressWriter) error {
	// using variable timeout approach from https://medium.com/@simonfrey/go-as-in-golang-standard-net-http-config-will-break-your-production-environment-1360871cb72b
	timer := time.AfterFunc(timeout*time.Second, func() {
		src.Close()
	})

	for {
		timer.Reset(timeout * time.Second)
		var reader io.Reader = src
		if tpw != nil {
			reader = io.TeeReader(src, tpw)
		}
		_, err := io.CopyN(dst, reader, blockSize)
		if err == io.EOF {
			// This is not an error in the common sense
			// io.EOF tells us, that we did read the complete body
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
