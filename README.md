# dolo

Golang module for file download. Dolo skips downloading if the file exists and has the same content length. Dolo supports an optional callback that notifies on progress. 

## Exported methods

- `NewClient(httpClient *http.Client, notify func(uint64, uint64))` - creates a new dolo client using the provided http.Client (with cookie jar, etc.) and an optional callback to notify on progress. Optional callback takes two values - downloaded bytes and total bytes.

- `Download(url *url.URL, dstDir string, overwrite bool)` - downloads a resource at the URL to the destination directory (original resource base path would be used for a filename). Overwrite would require download even if the destination file already exists and has the same content length.

