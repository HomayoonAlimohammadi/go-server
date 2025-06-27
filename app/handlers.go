package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
)

type handleFunc func(context.Context, *Request, io.Writer) error

func (s *server) rootGet(_ context.Context, _ *Request, w io.Writer) error {
	return httpResponse(w, http.StatusOK, nil, "")
}

func (s *server) handleNotFound(_ context.Context, _ *Request, w io.Writer) error {
	return httpResponse(w, http.StatusNotFound, nil, "")
}

func (s *server) echoGet(_ context.Context, req *Request, w io.Writer) error {
	echo := strings.TrimPrefix(req.Target, "/echo/")
	headers := NewResponseHeaders(req.Headers)
	headers.Set(HeaderContentType, ContentTypeTextPlain)

	encoder := encoderFromRequest(req)
	if encoder != nil {
		encoded, err := encoder.Encode([]byte(echo))
		if err != nil {
			err = fmt.Errorf("failed to encode response: %w", err)
			return httpResponse(w, http.StatusInternalServerError, headers, err.Error())
		}
		headers.Set(HeaderContentEncoding, EncodingGzip)
		return httpResponse(w, http.StatusOK, headers, encoded)
	}

	return httpResponse(w, http.StatusOK, headers, echo)
}

func (s *server) userAgentGet(_ context.Context, req *Request, w io.Writer) error {
	userAgent, _ := req.Headers.Get(HeaderUserAgent)
	headers := NewResponseHeaders(req.Headers)
	headers.Set(HeaderContentType, ContentTypeTextPlain)
	return httpResponse(w, http.StatusOK, headers, userAgent)
}

func (s *server) filesGet(_ context.Context, req *Request, w io.Writer) error {
	fileName := strings.TrimPrefix(req.Target, "/files/")
	b, err := os.ReadFile(path.Join(s.dir, fileName))
	if os.IsNotExist(err) {
		return httpResponse(w, http.StatusNotFound, nil, "")
	} else if err != nil {
		headers := NewResponseHeaders(req.Headers)
		headers.Set(HeaderContentType, ContentTypeTextPlain)
		return httpResponse(w, http.StatusInternalServerError, headers, err.Error())
	}
	headers := NewResponseHeaders(req.Headers)
	headers.Set(HeaderContentType, ContentTypeApplicationOctetStream)
	return httpResponse(w, http.StatusOK, headers, string(b))
}

func (s *server) filesPost(_ context.Context, req *Request, w io.Writer) error {
	fileName := strings.TrimPrefix(req.Target, "/files/")
	if err := os.WriteFile(path.Join(s.dir, fileName), req.Body, 0o644); err != nil {
		headers := NewResponseHeaders(req.Headers)
		headers.Set(HeaderContentType, ContentTypeTextPlain)
		return httpResponse(w, http.StatusInternalServerError, headers, err.Error())
	}
	return httpResponse(w, http.StatusCreated, nil, "")
}
