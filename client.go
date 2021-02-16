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
const maxRetires = 5

type Client struct {
	httpClient *http.Client
	retries    int
	notify     func(uint64, uint64)
}

func NewClient(httpClient *http.Client, retries int, notify func(uint64, uint64)) *Client {
	if retries < 1 {
		retries = 1
	} else if retries > maxRetires {
		retries = maxRetires
	}
	return &Client{httpClient: httpClient, retries: retries, notify: notify}
}

func (dolo *Client) Download(url *url.URL, dstDir string, overwrite bool) error {
	for rr := 0; rr < dolo.retries; rr++ {
		err := dolo.download(url, dstDir, overwrite)
		if err != nil {
			log.Println("dolo.Download: ", err)
		} else {
			return nil
		}
	}
	return nil
}

func (dolo *Client) download(url *url.URL, dstDir string, overwrite bool) error {
	resp, err := dolo.httpClient.Get(url.String())
	if err != nil {
		return err
	}

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
