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

func (dolo *Client) sameContentLength(url *url.URL, dstFilename string) (bool, error) {

	stat, err := os.Stat(dstFilename)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	if stat != nil {
		resp, err := dolo.httpClient.Head(url.String())
		if err != nil {
			return false, err
		}

		defer resp.Body.Close()

		if stat.Size() == resp.ContentLength {
			return true, nil
		}
	}

	return false, nil
}

func (dolo *Client) Download(url *url.URL, dstDir string, skipSame bool) error {

	dstFilename := filepath.Join(dstDir, path.Base(url.String()))

	sameCL, err := dolo.sameContentLength(url, dstFilename)
	if err != nil {
		return err
	}

	if sameCL && skipSame {
		log.Println("file already exists and has the same content length")
		return nil
	}

	resp, err := dolo.httpClient.Get(url.String())
	if err != nil {
		return err
	}

	defer resp.Body.Close()

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
