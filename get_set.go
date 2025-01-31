package dolo

import (
	"errors"
	"fmt"
	"github.com/boggydigital/nod"
	"net/http"
	"net/url"
	"time"
)

// GetSet downloads URLs and sets them to storage using indexes. E.g. URLs[index] is expected to be
// received and set by indexSetter(index). GetSet can use provided http.Client for authenticated requests.
// Finally, it supports reporting progress via provided nod.TotalProgressWriter object (optional).
func (cl *Client) GetSet(
	urls []*url.URL,
	indexSetter IndexSetter,
	tpw nod.TotalProgressWriter,
	force bool) map[int]error {

	indexErrors := make(map[int]error)

	if len(urls) > indexSetter.Len() {
		for i := indexSetter.Len() - 1; i < len(urls); i++ {
			indexErrors[i] = errors.New("not enough setters for this URL")
		}
	}

	if tpw != nil {
		tpw.Log("dolo.GetSet: %d URL(s)", len(urls))
		for i := 0; i < len(urls); i++ {
			tpw.Log("%d: %s", i, urls[i])
		}
	}

	errs := make(chan *IndexError)
	indexReadClosers := make(chan *indexReadCloser)
	results := make(chan *IndexResult)

	defer close(indexReadClosers)
	defer close(errs)
	defer close(results)

	total := len(urls)

	if total > 1 && tpw != nil {
		tpw.TotalInt(total)
	}

	ct := newIndexTracker(total)

	//work can be one of the following:
	//- requested items that need to be processed
	//- actively processing items for which we'd be waiting error or completion results
	for ct.hasWork() {

		//requested -> processing phase, start items processing pipeline while there is an opportunity
		//to add more: we have outstanding requested items and have availability to process items
		for ct.hasRequested() && ct.canProcess() {

			np := ct.processNext()
			if np == -1 {
				break
			}

			//performance optimization to support index setters that can provide an existence based on index.
			//for an example implementation check fileIndexSetter.Exists that checks if there is a local file
			//with a filename equal to fileIndexSetter.filenames[index]
			//additionally - if the source url is nil - skip it
			if (indexSetter.Exists(np) && !force) ||
				urls[np] == nil {
				ct.complete(np)
				if total > 1 && tpw != nil {
					tpw.Increment()
				}
				continue
			}

			modStr := ""
			if mod, err := indexSetter.FileModTime(np); err == nil && mod > 0 {
				modStr = time.Unix(mod, 0).Format(http.TimeFormat)
			}

			go cl.getReadCloser(urls[np], np, modStr, indexReadClosers, errs)
		}

		//break out of processing loop if there is no outstanding work we'd wait results for
		if !ct.hasProcessing() {
			break
		}

		select {
		case indErr := <-errs:
			indexErrors[indErr.index] = indErr.err
			if tpw != nil && indErr.err != nil {
				tpw.Error(indErr.err)
			}
			ct.complete(indErr.index)
			if total > 1 && tpw != nil {
				tpw.Increment()
			}
		case irc := <-indexReadClosers:
			go indexSetter.Set(irc.index, irc.readCloser, results, errs)
		case indRes := <-results:
			ct.complete(indRes.index)
			if total > 1 && tpw != nil {
				tpw.Increment()
			}
		}
	}

	return indexErrors
}

func (cl *Client) getReadCloser(
	u *url.URL,
	index int,
	modStr string,
	indexReadClosers chan *indexReadCloser,
	errors chan *IndexError) {

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		errors <- &IndexError{index, err}
		return
	}

	if cl.userAgent != "" {
		req.Header.Set("User-Agent", cl.userAgent)
	}

	if modStr != "" {
		req.Header.Set("If-Modified-Since", modStr)
	}

	if cl.requiresBasicAuth() {
		req.SetBasicAuth(cl.username, cl.password)
	}

	resp, err := cl.httpClient.Do(req)

	if err != nil ||
		resp.StatusCode == http.StatusNotModified {
		if resp != nil {
			defer resp.Body.Close()
		}
		errors <- &IndexError{index, err}
		return
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		errors <- &IndexError{
			index,
			fmt.Errorf("URL index %d response got status code %d", index, resp.StatusCode)}
		return
	}

	indexReadClosers <- &indexReadCloser{
		index:      index,
		readCloser: resp.Body,
	}
}
