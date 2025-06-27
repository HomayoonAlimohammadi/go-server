package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	HeaderAcceptEncoding  = "Accept-Encoding"
	HeaderContentLength   = "Content-Length"
	HeaderContentType     = "Content-Type"
	HeaderContentEncoding = "Content-Encoding"
	HeaderUserAgent       = "User-Agent"
	HeaderConnection      = "Connection"

	ContentTypeTextPlain              = "text/plain"
	ContentTypeApplicationOctetStream = "application/octet-stream"

	ConnectionKeepAlive = "keep-alive"
	ConnectionClose     = "close"
)

type Request struct {
	Method  string
	Target  string
	Version string
	Headers Headers
	Body    []byte
}

type Headers map[string]string

func NewResponseHeaders(reqHeaders Headers) Headers {
	copyHeaders := []string{
		HeaderConnection,
	}

	h := make(Headers)
	for _, cp := range copyHeaders {
		v, _ := reqHeaders.Get(cp)
		h.Set(cp, v)
	}

	return h
}

// Get retrieves the value of the given key if found.
// The given key is case-insensitive.
func (h Headers) Get(k string) (string, bool) {
	for hk, v := range h {
		if strings.EqualFold(hk, k) {
			return v, true
		}
	}
	return "", false
}

func (h Headers) Set(k, v string) {
	if k == "" {
		return
	} else if v == "" {
		delete(h, k)
		return
	}

	k = strings.TrimSpace(k)
	for hk := range h {
		if strings.EqualFold(hk, k) {
			h[hk] = v
			return
		}
	}
	h[http.CanonicalHeaderKey(k)] = v
}

func (r *Request) From(b []byte) error {
	s := string(b)
	parts := strings.Split(s, "\r\n")

	// Meta
	metaSegs := strings.Split(parts[0], " ")

	r.Method = strings.TrimSpace(metaSegs[0])
	r.Target = strings.TrimSpace(metaSegs[1])
	r.Version = strings.TrimSpace(metaSegs[2])

	// Headers
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	for _, l := range parts[1 : len(parts)-1] {
		if len(l) == 0 {
			continue
		}
		kv := strings.SplitN(l, ":", 2)
		k, v := kv[0], kv[1]
		r.Headers.Set(strings.TrimSpace(k), strings.TrimSpace(v))
	}

	// Body
	r.Body = []byte(parts[len(parts)-1])

	return nil
}

func (r *Request) String() string {
	s := fmt.Sprintf("%s\n%s\n%s\n\r\n", r.Method, r.Target, r.Version)
	for k, v := range r.Headers {
		s += fmt.Sprintf("%s:	%s\r\n", k, v)
	}
	s += fmt.Sprintf("\r\n%s", r.Body)
	return s
}

func httpResponse(w io.Writer, code int, headers Headers, body any) error {
	s := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\n",
		code,
		http.StatusText(code),
	)

	// Ensure we have a headers map
	if headers == nil {
		headers = make(Headers)
	}

	// Set Connection header to keep-alive by default if not already set
	// This allows clients to see that the server supports persistent connections
	if _, hasConnection := headers.Get(HeaderConnection); !hasConnection {
		headers.Set(HeaderConnection, ConnectionKeepAlive)
	}

	for h, v := range headers {
		s += fmt.Sprintf("%s: %s\r\n", h, v)
	}

	bodyLength := len(fmt.Sprintf("%s", body))
	if bodyLength > 0 {
		s += fmt.Sprintf("%s: %d\r\n\r\n%s", HeaderContentLength, bodyLength, body)
	}

	if _, err := w.Write([]byte(s)); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
}
