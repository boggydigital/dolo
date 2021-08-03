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
	"time"
)

const (
	downloadExt                 = ".download"
	minRetries                  = 1
	maxRetires                  = 6
	dirPerm         os.FileMode = 0755
	minDelaySeconds             = 5
	maxDelaySeconds             = 60
	blockSize                   = 32 * 1024
)

type Client struct {
	httpClient         *http.Client
	notify             func(uint64, uint64)
	attempts           int
	delayAttempts      int
	minSizeComplete    int64
	checkContentLength bool
	resumeDownloads    bool
	verbose            bool
}

type ClientOptions struct {
	Attempts           int
	DelayAttempts      int
	MinSizeComplete    int64
	CheckContentLength bool
	ResumeDownloads    bool
	Verbose            bool
}

type resourceStat struct {
	contentLength int64
	acceptRanges  bool
}

func enforceConstraints(val int, min, max int) int {
	if val < min {
		return min
	} else if val > max {
		return max
	}
	return val
}

func Defaults() *ClientOptions {
	return &ClientOptions{
		Attempts:        3,
		DelayAttempts:   5,
		ResumeDownloads: true,
		MinSizeComplete: 512,
	}
}

func NewClient(httpClient *http.Client, notify func(uint64, uint64), opts *ClientOptions) *Client {
	client := &Client{
		httpClient:         httpClient,
		notify:             notify,
		attempts:           enforceConstraints(opts.Attempts, minRetries, maxRetires),
		minSizeComplete:    opts.MinSizeComplete,
		delayAttempts:      enforceConstraints(opts.DelayAttempts, minDelaySeconds, maxDelaySeconds),
		checkContentLength: opts.CheckContentLength,
		resumeDownloads:    opts.ResumeDownloads,
		verbose:            opts.Verbose,
	}
	return client
}

func (dolo *Client) Download(url *url.URL, dstDir, dstFilename string) (network bool, err error) {
	for aa := 0; aa < dolo.attempts; aa++ {
		if aa > 0 {
			delaySec := dolo.delayAttempts * aa
			if dolo.verbose {
				log.Printf("dolo: delay next download attempt by %d seconds...\n", delaySec)
			}
			time.Sleep(time.Duration(delaySec) * time.Second)
			if dolo.verbose {
				log.Printf("dolo: retry download attempt %d/%d\n", aa+1, dolo.attempts)
			}
		}
		network, err = dolo.download(url, dstDir, dstFilename)
		if err != nil {
			if dolo.verbose {
				log.Println("dolo:", err)
			}
			return network, err
		} else {
			return network, nil
		}
	}
	return network, nil
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

func extractStat(resp *http.Response) *resourceStat {
	stat := &resourceStat{
		contentLength: resp.ContentLength,
	}

	acceptRangesHeaders := resp.Header.Values("Accept-Ranges")
	for _, arh := range acceptRangesHeaders {
		if arh != "" && arh != "none" {
			stat.acceptRanges = true
		}
	}

	return stat
}

func attemptSrcHead(url *url.URL, httpClient *http.Client) (stat *resourceStat, err error) {
	resp, err := httpClient.Head(url.String())
	if err != nil {
		return stat, err
	}

	defer resp.Body.Close()

	return extractStat(resp), nil
}

func (dolo *Client) srcHead(url *url.URL) (stat *resourceStat, err error) {

	cont := true
	attempt := 0

	for cont {
		attempt++
		if attempt > 1 {
			delay := (time.Duration)((attempt - 1) * dolo.delayAttempts)
			if dolo.verbose {
				log.Printf("dolo: delay source content info request attempt by %ds...\n", delay)
			}
			time.Sleep(delay * time.Second)
			if dolo.verbose {
				log.Printf("dolo: source content info request attempt %d/%d\n", attempt, dolo.attempts)
			}
		}
		stat, err = attemptSrcHead(url, dolo.httpClient)
		if err != nil {
			return stat, err
		}

		if !stat.acceptRanges ||
			attempt == dolo.attempts ||
			(stat.acceptRanges &&
				stat.contentLength > 0) {
			cont = false
		}
	}

	if stat.acceptRanges &&
		stat.contentLength == 0 {
		if dolo.verbose {
			log.Printf("dolo: accept-ranges = bytes and content-length = 0 -> download restart")
		}
		stat.acceptRanges = false
	}

	return stat, err
}

// download implements file downloader that checks for existing file,
// can optionally compare content length to verify that content has
// changed.
// download detects partial downloads (.download files) and would
// attempt to continue from the last position.
func (dolo *Client) download(url *url.URL, dstDir, dstFilename string) (network bool, err error) {

	if err := os.MkdirAll(dstDir, dirPerm); err != nil {
		return false, err
	}

	if dstFilename == "" {
		dstFilename = path.Base(url.String())
	}

	dstFilename = filepath.Join(dstDir, dstFilename)

	downloadFilename := dstFilename + downloadExt

	// check if destination file (not .download!) has positive size
	// and optionally, same content length as the source resource.
	// This is the first opportunity to return early without doing any work
	network, exists, stat, err := dolo.checkDstFile(url, dstFilename)
	if err != nil {
		return network, err
	}
	if exists {
		if dolo.verbose {
			log.Println("dolo: skip downloading existing file")
		}
		return network, nil
	}

	// we've established that destination file either doesn't exist or
	// has different content length. In both cases we need to re-download.
	// before we do that - check if .download file exists and attempt
	// resuming download.
	net, req, downloadFile, err := dolo.requestAndFile(url, downloadFilename, stat)
	if err != nil {
		return network, err
	}

	network = network || net

	if downloadFile == nil {
		downloadFile, err = os.Create(downloadFilename)
		if err != nil {
			return network, err
		}
	}
	defer downloadFile.Close()

	network = true
	if err = dolo.downloadRequest(req, downloadFile); err != nil {
		return network, err
	}

	osStat, err := os.Stat(downloadFilename)
	if err != nil {
		return network, err
	}
	if osStat.Size() > 0 {
		return network, os.Rename(downloadFilename, dstFilename)
	}

	return network, nil
}

func (dolo *Client) downloadRequest(
	srcReq *http.Request,
	dstFile *os.File) error {

	resp, err := dolo.httpClient.Do(srcReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	prg := &progress{
		total: uint64(resp.ContentLength)}

	//only use notifications for files that are larger than 32K,
	//as io.Copy currently has this set for a single copy attempt
	if resp.ContentLength > blockSize {
		prg.notify = dolo.notify
		if dolo.notify != nil {
			dolo.notify(0, uint64(resp.ContentLength))
		}
	}

	// using variable timeout approach from https://medium.com/@simonfrey/go-as-in-golang-standard-net-http-config-will-break-your-production-environment-1360871cb72b
	timer := time.AfterFunc(5*time.Second, func() {
		resp.Body.Close()
	})

	for {
		//We reset the timer, for the variable time
		timer.Reset(1 * time.Second)
		_, err = io.CopyN(dstFile, io.TeeReader(resp.Body, prg), blockSize)
		if err == io.EOF {
			// This is not an error in the common sense
			// io.EOF tells us, that we did read the complete body
			break
		} else if err != nil {
			//You should do error handling here
			return err
		}
	}

	return nil
}

func (dolo *Client) requestAndFile(
	url *url.URL,
	filename string,
	stat *resourceStat) (network bool, req *http.Request, file *os.File, err error) {

	req = &http.Request{
		Method: http.MethodGet,
		URL:    url,
	}

	if !dolo.resumeDownloads {
		return network, req, file, nil
	}

	downloadSize, err := dolo.dstSize(filename)
	if err != nil {
		return network, nil, nil, err
	}

	if downloadSize > 0 {
		if stat == nil || stat.contentLength == -1 {
			stat, err = dolo.srcHead(url)
			network = true
			if err != nil {
				return network, req, file, err
			}
		}

		if stat.acceptRanges {
			if dolo.verbose {
				log.Printf("dolo: resume download, bytes %d to %d", downloadSize, stat.contentLength-1)
			}
			if req.Header == nil {
				req.Header = make(map[string][]string, 0)
			}
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", downloadSize, stat.contentLength-1))
			file, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return network, req, file, err
			}
		}
	}

	return network, req, file, nil
}

func (dolo *Client) checkDstFile(url *url.URL, filename string) (network bool, exists bool, stat *resourceStat, err error) {
	dstSize, err := dolo.dstSize(filename)
	if err != nil {
		return network, exists, stat, err
	}

	// not using zero in case server
	exists = dstSize > dolo.minSizeComplete

	if exists {
		if !dolo.checkContentLength {
			if dolo.verbose {
				log.Println("dolo: destination file exists, " +
					"has positive size, content length check skipped")
			}
			return network, exists, stat, err
		} else {
			stat, err = dolo.srcHead(url)
			network = true
			if err != nil {
				return network, exists, stat, err
			}

			if stat.contentLength > 0 && dstSize == stat.contentLength {
				if dolo.verbose {
					log.Println("dolo: destination file exists, " +
						"matches source content length")
				}
				return network, exists, stat, err
			}
		}
	}

	return network, exists, stat, err
}
