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
	Set(int, io.ReadCloser, chan io.Closer, chan error)
	Len() int
}

func GetSetOne(
	url *url.URL,
	index int,
	indexSetter IndexSetter,
	httpClient *http.Client) error {

	errors := make(chan error)
	writeClosers := make(chan io.Closer)

	defer close(errors)
	defer close(writeClosers)

	resp, err := httpClient.Get(url.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	go indexSetter.Set(index, resp.Body, writeClosers, errors)

	select {
	case err := <-errors:
		return err
	case wc := <-writeClosers:
		if err := wc.Close(); err != nil {
			return err
		}
	}

	return nil
}

func GetSetMany(
	urls []*url.URL,
	indexSetter IndexSetter,
	httpClient *http.Client,
	tpw nod.TotalProgressWriter) error {

	if len(urls) != indexSetter.Len() {
		return fmt.Errorf("unequal number of urls and writers")
	}

	errors := make(chan error)
	indexReadClosers := make(chan *indexReadCloser)
	writeClosers := make(chan io.Closer)

	defer close(indexReadClosers)
	defer close(errors)
	defer close(writeClosers)

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
			go indexSetter.Set(irc.index, irc.readCloser, writeClosers, errors)
		case wc := <-writeClosers:
			if err := wc.Close(); err != nil {
				return tpw.EndWithError(err)
			}
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
