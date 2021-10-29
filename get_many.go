package dolo

import (
	"fmt"
	"github.com/boggydigital/nod"
	"io"
	"net/http"
	"net/url"
	"runtime"
)

type indexReadCloser struct {
	index      int
	readCloser io.ReadCloser
}

type IndexSetter interface {
	Set(index int, src io.ReadCloser) error
	Len() int
}

func GetMany(
	urls []*url.URL,
	indexSetter IndexSetter,
	httpClient *http.Client,
	tpw nod.TotalProgressWriter) error {

	if len(urls) != indexSetter.Len() {
		return fmt.Errorf("unequal number of urls and writers")
	}

	errors := make(chan error)
	indexReadClosers := make(chan *indexReadCloser)

	defer close(indexReadClosers)
	defer close(errors)

	tpw.TotalInt(len(urls))

	current, total := 0, len(urls)
	concurrentPages := runtime.NumCPU()

	remaining := total - current
	for remaining > 0 {

		for i := 0; i < concurrentPages; i++ {
			current++
			if current > total {
				break
			}
			if total > 1 {
				tpw.Increment()
			}
			index := current - 1
			go getReadCloser(urls[index], index, httpClient, indexReadClosers, errors)
		}
		concurrentPages = 0

		select {
		case err := <-errors:
			return tpw.EndWithError(err)
		case irc := <-indexReadClosers:
			src := irc.readCloser
			if err := indexSetter.Set(irc.index, src); err != nil {
				return tpw.EndWithError(err)
			}
			irc.readCloser.Close()
			remaining--
			concurrentPages++
		}
	}

	return nil
}

func getReadCloser(
	u *url.URL,
	index int,
	httpClient *http.Client,
	indexReadClosers chan *indexReadCloser,
	errors chan error) {

	resp, err := httpClient.Get(u.String())
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		errors <- err
		return
	}

	indexReadClosers <- &indexReadCloser{
		index:      index,
		readCloser: resp.Body,
	}
}
