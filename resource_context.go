package dolo

import (
	"fmt"
	"github.com/boggydigital/nod"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	downloadExt             = ".download"
	dirPerm     os.FileMode = 0755
)

type resourceContext struct {
	checkedRemoteStat     bool
	remoteContentLength   int64
	remoteAcceptRanges    bool
	localSize             int64
	localDir              string
	localFilename         string
	localFileExists       bool
	partialDownloadExists bool
	partialDownloadSize   int64
}

func NewResourceContext(u *url.URL, pathParts ...string) *resourceContext {
	rsc := &resourceContext{}

	//fill local directory and filename using the following rules:
	//1) if nothing was specified - use source URL path base
	//2) if only one component was specified:
	//2.1) if that component has an extension - use component as local filename
	//2.2) otherwise use component as local directory
	//3) if more than one component was specified - join them to get directory and filename
	if len(pathParts) == 0 {
		rsc.localFilename = filepath.Base(u.Path)
	} else if len(pathParts) == 1 {

		if strings.HasSuffix(pathParts[0], filepath.Base(u.Path)) {
			rsc.localDir, rsc.localFilename = filepath.Split(pathParts[0])
		} else if filepath.Ext(pathParts[0]) != "" {
			rsc.localDir, rsc.localFilename = filepath.Split(pathParts[0])
		} else {
			rsc.localDir = pathParts[0]
			rsc.localFilename = filepath.Base(u.String())
		}

	} else {
		rsc.localDir, rsc.localFilename = filepath.Split(filepath.Join(pathParts...))
	}

	return rsc
}

func (rsc *resourceContext) downloadPath() string {
	return filepath.Join(rsc.localDir, rsc.localFilename)
}

func (rsc *resourceContext) partialDownloadPath() string {
	return rsc.downloadPath() + downloadExt
}

func (rsc *resourceContext) createLocalDirs() error {
	if rsc.localDir == "" {
		return nil
	}

	if _, err := os.Stat(rsc.localDir); err == nil || !os.IsNotExist(err) {
		return err
	} else {
		if err := os.MkdirAll(rsc.localDir, dirPerm); err != nil {
			return err
		}
	}

	return nil
}

func (rsc *resourceContext) shouldTryResuming(resumeDownloads bool) bool {
	return resumeDownloads &&
		rsc.checkedRemoteStat &&
		rsc.remoteAcceptRanges &&
		rsc.partialDownloadExists &&
		rsc.partialDownloadSize > 0
}

func (rsc *resourceContext) setRangeHeader(req *http.Request, tpw nod.TotalProgressWriter) {

	req.Header.Set(
		"Range",
		fmt.Sprintf(
			"bytes=%d-",
			rsc.partialDownloadSize))

	if tpw != nil {
		if rsc.remoteContentLength > 0 {
			tpw.Total(uint64(rsc.remoteContentLength))
		}
		tpw.Current(uint64(rsc.partialDownloadSize))
	}
}

func (rsc *resourceContext) openLocalFile(resumeDownloads bool) (*os.File, error) {
	pdp := rsc.partialDownloadPath()
	if rsc.shouldTryResuming(resumeDownloads) {
		return os.OpenFile(pdp, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	}
	return os.Create(pdp)
}

func (rsc *resourceContext) completePartialDownload() error {
	pdp := rsc.partialDownloadPath()
	if stat, err := os.Stat(pdp); err != nil {
		return err
	} else {
		if stat.Size() > 0 {
			return os.Rename(pdp, rsc.downloadPath())
		}
	}
	return nil
}
