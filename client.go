package dolo

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
)

const downloadExt = ".download"
const maxRetires = 5

type Client struct {
	httpClient         *http.Client
	notify             func(uint64, uint64)
	retries            int
	checkContentLength bool
	resumeDownloads    bool
	verbose            bool
}

type ClientOptions struct {
	Retries            int
	CheckContentLength bool
	ResumeDownloads    bool
	Verbose            bool
}

func NewClient(httpClient *http.Client, notify func(uint64, uint64), opts *ClientOptions) *Client {
	client := &Client{
		httpClient:         httpClient,
		notify:             notify,
		retries:            opts.Retries,
		checkContentLength: opts.CheckContentLength,
		resumeDownloads:    opts.ResumeDownloads,
		verbose:            opts.Verbose,
	}
	if client.retries < 1 {
		client.retries = 1
	} else if client.retries > maxRetires {
		client.retries = maxRetires
	}
	return client
}

func (dolo *Client) Download(url *url.URL, dstDir string) error {
	for rr := 0; rr < dolo.retries; rr++ {
		if rr > 0 {
			log.Printf("retrying download, attempt %d/%d\n", rr+1, dolo.retries)
		}
		err := dolo.download(url, dstDir)
		if err != nil {
			log.Println("dolo.Download: ", err)
		} else {
			return nil
		}
	}
	return nil
}

func (dolo *Client) dstSize(dstFilename string) (int64, error) {
	stat, err := os.Stat(dstFilename)
	if err != nil && !os.IsNotExist(err) {
		return -1, err
	}

	if stat != nil {
		return stat.Size(), nil
	}

	return -1, nil
}

func (dolo *Client) srcStat(url *url.URL) (contentLength int64, acceptRanges bool, err error) {
	resp, err := dolo.httpClient.Head(url.String())
	if err != nil {
		return -1, false, err
	}

	defer resp.Body.Close()

	contentLength = resp.ContentLength
	acceptRangesHeaders := resp.Header.Values("Accept-Ranges")
	for _, arh := range acceptRangesHeaders {
		if arh != "" && arh != "none" {
			acceptRanges = true
		}
	}

	return contentLength, acceptRanges, err
}

// download implements file downloader that checks for existing file,
// can optionally compare content length to verify that content has
// changed.
// download detects partial downloads (.download files) and would
// attempt to continue from the last position.
func (dolo *Client) download(url *url.URL, dstDir string) error {

	dstFilename := filepath.Join(dstDir, path.Base(url.String()))
	downloadFilename := dstFilename + downloadExt
	var contentLength int64 = -1
	acceptRanges := false

	// check if destination file (not .download!) has positive size
	// and optionally, same content length as the source resource.
	// This is the first opportunity to return early without doing any work
	dstSize, err := dolo.dstSize(dstFilename)
	if err != nil {
		return err
	}

	if dstSize > 0 {
		if !dolo.checkContentLength {
			// log: destination file exists, has positive size and
			// we were not requested to check the size
			log.Println("destination file exists, has positive size and we were not requested to check the size")
			return nil
		} else {
			contentLength, acceptRanges, err = dolo.srcStat(url)
			if err != nil {
				return err
			}

			if contentLength > 0 && dstSize == contentLength {
				log.Println("destination file exists, has positive size and same content length as the source resource")
				// log: destination file exists, has positive size and
				// same content length as the source resource
				return nil
			}
		}
	}

	req := &http.Request{
		Method: http.MethodGet,
		URL:    url,
	}
	var dstFile *os.File

	// we've established that destination file either doesn't exist or
	// has different content length. In both cases we need to re-download.
	// before we do that - check if .download file exists and attempt
	// resuming download.
	if dolo.resumeDownloads {
		downloadSize, err := dolo.dstSize(downloadFilename)
		if err != nil {
			return err
		}

		if downloadSize > 0 {
			if contentLength == -1 {
				contentLength, acceptRanges, err = dolo.srcStat(url)
				if err != nil {
					return err
				}
			}

			if acceptRanges {
				// set req to download range
				// log: attempting resuming download...
				log.Printf("attempting to resume download, bytes %d to %d", downloadSize, contentLength-1)
				if req.Header == nil {
					req.Header = make(map[string][]string, 0)
				}
				req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", downloadSize, contentLength-1))
				dstFile, err = os.OpenFile(downloadFilename, os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					return err
				}
			}
		}
	}

	// no attempt to avoid doing a full download succeeded.
	resp, err := dolo.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if dstFile == nil {
		dstFile, err = os.Create(downloadFilename)
		if err != nil {
			return err
		}
	}

	defer dstFile.Close()

	prg := &progress{
		total:  uint64(resp.ContentLength),
		notify: dolo.notify}

	if _, err = io.Copy(dstFile, io.TeeReader(resp.Body, prg)); err != nil {
		return err
	}

	return os.Rename(downloadFilename, dstFilename)
}
