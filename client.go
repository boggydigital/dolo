package dolo

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
)

const downloadExt = ".download"

type Client struct {
	httpClient *http.Client
	notify     func(uint64, uint64)
}

func NewClient(httpClient *http.Client, notify func(uint64, uint64)) *Client {
	return &Client{httpClient: httpClient, notify: notify}
}

func (dolo *Client) DownloadUrl(url *url.URL, dstDir string, overwrite bool) error {
	resp, err := dolo.httpClient.Get(url.String())
	if err != nil {
		return err
	}

	return dolo.downloadResponse(resp, url, dstDir, overwrite)
}

func (dolo *Client) DownloadReq(req *http.Request, dstDir string, overwrite bool) error {
	resp, err := dolo.httpClient.Do(req)
	if err != nil {
		return err
	}
	return dolo.downloadResponse(resp, req.URL, dstDir, overwrite)
}

func (dolo *Client) downloadResponse(resp *http.Response, url *url.URL, dstDir string, overwrite bool) error {
	defer resp.Body.Close()

	dstFilename := filepath.Join(dstDir, path.Base(url.String()))

	if !overwrite {
		stat, err := os.Stat(dstFilename)
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		if stat != nil {
			if stat.Size() == resp.ContentLength {
				log.Println("file already exists and has the same content length")
				return nil
			}
		}
	}

	dstFile, err := os.Create(dstFilename + downloadExt)
	if err != nil {
		return err
	}

	defer dstFile.Close()

	prg := &progress{
		total:  uint64(resp.ContentLength),
		notify: dolo.notify}

	if _, err = io.Copy(dstFile, io.TeeReader(resp.Body, prg)); err != nil {
		return err
	}

	if err = os.Rename(dstFilename+downloadExt, dstFilename); err != nil {
		return err
	}

	return nil
}
