package dolo

import (
	"fmt"
	"github.com/boggydigital/nod"
	"io"
	"net/http"
	"net/url"
)

type indexReadCloser struct {
	index      int
	readCloser io.ReadCloser
}

type IndexSetter interface {
	Set(int, io.ReadCloser, chan bool, chan error)
	Exists(int) bool
	Len() int
}

//GetSet downloads URLs and sets them to storage using indexes. E.g. URLs[index] is expected to be
//received and set by indexSetter(index). GetSet can use provided http.Client for authenticated requests.
//Finally, it supports reporting progress via provided nod.TotalProgressWriter object (optional).
func GetSet(
	urls []*url.URL,
	indexSetter IndexSetter,
	httpClient *http.Client,
	tpw nod.TotalProgressWriter) error {

	if len(urls) != indexSetter.Len() {
		return fmt.Errorf("unequal number of urls and writers")
	}

	tpw.Log("dolo.GetSet: starting to process %d URL(s)", len(urls))
	for i := 0; i < len(urls); i++ {
		tpw.Log("%d: %s", i, urls[i])
	}

	errors := make(chan error)
	indexReadClosers := make(chan *indexReadCloser)
	completion := make(chan bool)

	defer close(indexReadClosers)
	defer close(errors)
	defer close(completion)

	ct := newConcurrencyCounter(len(urls), tpw)

	for ct.hasRemaining() {

		//performance optimization to support index setters that can provide an existence based on index.
		//for an example implementation check fileIndexSetter.Exists that checks if there is a local file
		//with a filename equal to fileIndexSetter.filenames[index]
		if indexSetter.Exists(ct.current()) {
			ct.complete()
			continue
		}

		for i := 0; i < ct.allowedConcurrent(); i++ {
			if !ct.canSchedule() {
				break
			}

			go getReadCloser(urls[ct.current()], ct.current(), httpClient, indexReadClosers, errors)

			ct.scheduleNext()
		}
		ct.flushRemainingConcurrent()

		select {
		case err := <-errors:
			tpw.Error(err)
			ct.complete()
		case irc := <-indexReadClosers:
			go indexSetter.Set(irc.index, irc.readCloser, completion, errors)
		case <-completion:
			ct.complete()
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
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errors <- fmt.Errorf("URL index %d response got status code %d", index, resp.StatusCode)
		if resp != nil {
			resp.Body.Close()
		}
		return
	}

	indexReadClosers <- &indexReadCloser{
		index:      index,
		readCloser: resp.Body,
	}
}
