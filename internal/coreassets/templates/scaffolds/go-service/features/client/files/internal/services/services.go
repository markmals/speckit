// Package services holds clients for the upstream HTTP APIs this service depends
// on, plus the shared plumbing they use: a bounded timeout, JSON decoding, typed
// non-2xx errors, and URL-error sanitization (so a request URL — which may carry
// an API key in its query string — never leaks into logs). example.go shows the
// pattern; add a file (or subpackage) per upstream you call.
package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultTimeout bounds every upstream call — never let a slow dependency wedge a
// request goroutine.
const DefaultTimeout = 15 * time.Second

// APIError is a non-2xx response from an upstream service.
type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("upstream returned %d: %s", e.Status, e.Body)
}

// GetJSON executes req with hc and decodes a 2xx JSON body into dest. A non-2xx
// response becomes an *APIError (with a bounded snippet of the body); a transport
// error is sanitized first (see SanitizeURLError) so logging it can't leak a
// secret-bearing URL.
func GetJSON(hc *http.Client, req *http.Request, dest any) error {
	resp, err := hc.Do(req)
	if err != nil {
		return SanitizeURLError(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return &APIError{Status: resp.StatusCode, Body: strings.TrimSpace(string(body))}
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

// SanitizeURLError unwraps a *url.Error — whose message embeds the full request
// URL, query string and all — to its inner cause, so an API key passed as a query
// parameter can't end up in a log line. Non-url.Error values pass through.
func SanitizeURLError(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		return ue.Err
	}
	return err
}
