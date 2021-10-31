package dolo

import (
	"fmt"
	"github.com/boggydigital/nod"
	"io"
	"net/http"
	"net/url"
	"runtime"
)

const maxConcurrentDownloads = 8

type counter struct {
	current   int
	total     int
	remaining int
}

type indexReadCloser struct {
	index      int
	readCloser io.ReadCloser
}

type IndexSetter interface {
	Set(int, io.ReadCloser, chan bool, chan error)
	Exists(int) bool
	Len() int
}

func GetSet(
	urls []*url.URL,
	indexSetter IndexSetter,
	httpClient *http.Client,
	tpw nod.TotalProgressWriter) error {

	if len(urls) != indexSetter.Len() {
		return fmt.Errorf("unequal number of urls and writers")
	}

	errors := make(chan error)
	indexReadClosers := make(chan *indexReadCloser)
	completion := make(chan bool)

	defer close(indexReadClosers)
	defer close(errors)
	defer close(completion)

	if len(urls) > 1 {
		tpw.TotalInt(len(urls))
	}

	current, total := 0, len(urls)
	concurrentPages := runtime.NumCPU()
	if concurrentPages > maxConcurrentDownloads {
		concurrentPages = maxConcurrentDownloads
	}

	remaining := total - current
	for remaining > 0 {

		if indexSetter.Exists(current) {
			current++
			remaining--
			if total > 1 {
				tpw.Increment()
			}
			continue
		}

		for i := 0; i < concurrentPages; i++ {
			current++
			if current > total {
				break
			}
			index := current - 1
			go getReadCloser(urls[index], index, httpClient, indexReadClosers, errors)
		}
		concurrentPages = 0

		select {
		case err := <-errors:
			tpw.Error(err)
			remaining--
			concurrentPages++
			if total > 1 {
				tpw.Increment()
			}
		case irc := <-indexReadClosers:
			go indexSetter.Set(irc.index, irc.readCloser, completion, errors)
		case <-completion:
			remaining--
			concurrentPages++
			if total > 1 {
				tpw.Increment()
			}
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
