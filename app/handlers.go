package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
)

type handleFunc func(context.Context, *Request, io.Writer) error

func (s *server) handleRoot(_ context.Context, _ *Request, w io.Writer) error {
	return httpResponse(w, http.StatusOK, "", "")
}

func (s *server) handleNotFound(_ context.Context, _ *Request, w io.Writer) error {
	return httpResponse(w, http.StatusNotFound, "", "")
}

func (s *server) handleEcho(_ context.Context, req *Request, w io.Writer) error {
	echo := strings.TrimPrefix(req.Target, "/echo/")
	return httpResponse(w, http.StatusOK, "text/plain", echo)
}

func (s *server) handleUserAgent(_ context.Context, req *Request, w io.Writer) error {
	return httpResponse(w, http.StatusOK, "text/plain", req.Headers["User-Agent"])
}

func (s *server) handleFiles(_ context.Context, req *Request, w io.Writer) error {
	fileName := strings.TrimPrefix(req.Target, "/files/")
	b, err := os.ReadFile(path.Join(s.dir, fileName))
	if os.IsNotExist(err) {
		return httpResponse(w, http.StatusNotFound, "", "")
	} else if err != nil {
		return httpResponse(w, http.StatusInternalServerError, "text/plain", err.Error())
	}
	return httpResponse(w, http.StatusOK, "application/octet-stream", string(b))
}
