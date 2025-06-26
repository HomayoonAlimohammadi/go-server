package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Request struct {
	Method  string
	Target  string
	Version string
	Headers map[string]string
	Body    []byte
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
		r.Headers[strings.TrimSpace(k)] = strings.TrimSpace(v)
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

func httpResponse(w io.Writer, code int, body string) error {
	s := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\n",
		code,
		http.StatusText(code),
	)

	if len(body) > 0 {
		s += fmt.Sprintf("Content-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(body), body)
	}

	if _, err := w.Write([]byte(s)); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
}
