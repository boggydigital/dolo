package dolo

import (
	"fmt"
	"github.com/boggydigital/nod"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

// Download gets remote resource, attempting to resume existing partial downloads if the previous
// attempt got interrupted. Download supports nod progress reporting.
// Download will use provided path parts (see NewResourceContext for file name, directory rules),
// and if no parts were specified - will download to the nextAvailable working directory and use source
// URL path base for a filename.
func (dc *Client) Download(u *url.URL, tpw nod.TotalProgressWriter, pathParts ...string) error {

	//process path parts and set local directory and filename using conventional rules
	rsc := NewResourceContext(u, pathParts...)

	//make sure that full local directory path exists
	if err := rsc.createLocalDirs(); err != nil {
		return err
	}

	//first opportunity to return early - check if local file (not .download!) exists:
	//- file at local path must exist
	//- it must have positive size
	//- (if set in dolo client options) it should have size matching content length of the remote resourceContext
	if err := dc.checkLocalFile(u, rsc); err != nil {
		return err
	}
	if rsc.localFileExists {
		return nil
	}

	//check if a partial download exists (e.g. filename.ext.download)
	//add information for it in the resourceContext context
	if err := dc.checkPartialDownload(u, rsc); err != nil {
		return err
	}

	//at this point we've got all resourceContext context information available
	//and can start a remote resourceContext request
	if err := dc.downloadResource(u, rsc, tpw); err != nil {
		return err
	}

	//after downloading remote resource - complete partial download by renaming the file
	return rsc.completePartialDownload()
}

// checkRemoteStat performs HEAD request and extracts Accept-Ranges and
// Content-Length headers information
func (dc *Client) checkRemoteStat(url *url.URL, rsc *resourceContext) error {

	resp, err := dc.httpClient.Head(url.String())
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	acceptRangesHeaders := resp.Header.Values("Accept-Ranges")
	for _, arh := range acceptRangesHeaders {
		if arh != "" && arh != "none" {
			rsc.remoteAcceptRanges = true
		}
	}

	contentLengthHeaders := resp.Header.Values("Content-Length")
	if len(contentLengthHeaders) == 1 {
		cl, err := strconv.Atoi(contentLengthHeaders[0])
		if err != nil {
			return err
		}
		rsc.remoteContentLength = int64(cl)
	}

	rsc.checkedRemoteStat = true
	rsc.remoteAcceptRanges = rsc.remoteAcceptRanges && rsc.remoteContentLength > 0

	return nil
}

// checkLocalFile adds information about local download - file that would exist
// in the same directory and have the same name as requested download resource.
// If requested checkLocalFile will compare existing file size to remote resource
// content length.
func (dc *Client) checkLocalFile(u *url.URL, rsc *resourceContext) error {

	if stat, err := os.Stat(rsc.downloadPath()); err != nil && !os.IsNotExist(err) {
		// error reading file information, and it's not "file doesn't exist"
		return err
	} else if err == nil {
		// fill local filename size
		rsc.localFileExists = true
		rsc.localSize = stat.Size()
	} // no need to handle os.IsNotExist(err) as we won't do anything in that case

	//if local file exists, but has zero length - consider that it doesn't exist
	//as that suggests server returned zero length (otherwise valid) response
	rsc.localFileExists = rsc.localSize > 0

	//unless asked to check remote content length and file exists - return early
	if !dc.checkContentLength ||
		!rsc.localFileExists {
		return nil
	}

	if err := dc.checkRemoteStat(u, rsc); err != nil {
		return err
	}

	//when asked to check remote content length, local file is considered existing
	//only when it's size matches remote content length
	rsc.localFileExists = rsc.remoteContentLength > 0 && rsc.localSize == rsc.remoteContentLength

	return nil
}

// checkPartialDownload adds information about partial download - file that would exist
// in the same directory and have same name as requested download resource (plus .download extension).
// checkPartialDownload will also checkRemoteStat, unless already done, because we need to know
// if the server supports resuming downloads before attempting that.
func (dc *Client) checkPartialDownload(u *url.URL, rsc *resourceContext) error {

	//don't attempt resuming partial downloads if not requested in options, or we checked
	//remote stats and the source doesn't accept ranges (we won't be able to resume)
	if !dc.resumeDownloads ||
		(rsc.checkedRemoteStat && !rsc.remoteAcceptRanges) {
		return nil
	}

	if stat, err := os.Stat(rsc.partialDownloadPath()); os.IsNotExist(err) {
		//partial download doesn't exist, won't attempt resuming download
		return nil
	} else if err != nil {
		return err
	} else {
		rsc.partialDownloadExists = true
		rsc.partialDownloadSize = stat.Size()
	}

	if !rsc.checkedRemoteStat {
		if err := dc.checkRemoteStat(u, rsc); err != nil {
			return err
		}
	}

	return nil
}

// downloadResource requests remote source response and copies bytes into partial download file.
// downloadResource will resume download when we've established it's feasible, however if the server
// would not honor that request - it'll restart file download. downloadResource supports nod progress reporting.
func (dc *Client) downloadResource(u *url.URL, rsc *resourceContext, tpw nod.TotalProgressWriter) error {

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}

	//when resourceContext context indicates that we should attempt downloading existing partial download -
	//add Range: bytes={from}- header
	if rsc.shouldTryResuming(dc.resumeDownloads) {
		rsc.setRangeHeader(req, tpw)
	}

	if dc.userAgent != "" {
		req.Header.Set("User-Agent", dc.userAgent)
	}

	resp, err := dc.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("error status code %d", resp.StatusCode)
	}

	resumeDownloads := dc.resumeDownloads

	switch resp.StatusCode {
	case http.StatusPartialContent:
		// nothing special needs to be done, capturing this case for visibility
	case http.StatusOK:
		if tpw != nil {
			tpw.Current(0)
		}
		resumeDownloads = false
	}

	file, err := rsc.openLocalFile(resumeDownloads)
	if err != nil {
		return err
	}

	if tpw != nil &&
		resp.ContentLength > 0 {
		//to set total - start with content length we got from the server
		total := resp.ContentLength
		//and if we're attempting to resume download - add existing partial download size
		//since server would respond with content length of the remaining part of the file
		if rsc.shouldTryResuming(dc.resumeDownloads) {
			total += rsc.partialDownloadSize
		}
		tpw.Total(uint64(total))
	}

	if err := CopyWithProgress(file, resp.Body, tpw); err != nil {
		return err
	}

	return file.Close()
}
