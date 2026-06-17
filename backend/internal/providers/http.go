package providers

import (
	"io"
	"net/http"
	"time"
)

// httpClient is the shared client for all outbound provider fetches.
// A timeout is mandatory so a slow or malicious upstream cannot block a
// goroutine indefinitely.
var httpClient = &http.Client{Timeout: 15 * time.Second}

// maxFetchBytes caps how much of an upstream response body we read, to bound
// memory use on hostile or oversized responses.
const maxFetchBytes = 5 << 20 // 5 MB

// limitedBody wraps a response body so reads are capped at maxFetchBytes while
// the underlying body is still closed by the caller's defer.
func limitedBody(body io.ReadCloser) io.ReadCloser {
	return struct {
		io.Reader
		io.Closer
	}{io.LimitReader(body, maxFetchBytes), body}
}
