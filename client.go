package dolo

import (
	"net/http"
)

const (
	DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Safari/605.1.15"
)

type Client struct {
	httpClient         *http.Client
	userAgent          string
	checkContentLength bool
	resumeDownloads    bool
}

type ClientOptions struct {
	UserAgent          string
	CheckContentLength bool
	ResumeDownloads    bool
}

func Defaults() *ClientOptions {
	return &ClientOptions{
		UserAgent:          DefaultUserAgent,
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
		userAgent:          opts.UserAgent,
	}
	return client
}

var DefaultClient = NewClient(http.DefaultClient, Defaults())
