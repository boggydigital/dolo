package dolo

import (
	"net/http"
)

type Client struct {
	httpClient         *http.Client
	checkContentLength bool
	resumeDownloads    bool
}

type ClientOptions struct {
	CheckContentLength bool
	ResumeDownloads    bool
}

func Defaults() *ClientOptions {
	return &ClientOptions{
		CheckContentLength: false,
		ResumeDownloads:    true,
	}
}

func NewClient(httpClient *http.Client, opts *ClientOptions) *Client {
	if opts == nil {
		opts = Defaults()
	}
	client := &Client{
		httpClient:         httpClient,
		checkContentLength: opts.CheckContentLength,
		resumeDownloads:    opts.ResumeDownloads,
	}
	return client
}

var DefaultClient = NewClient(http.DefaultClient, Defaults())
