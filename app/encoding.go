package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"log"
	"strings"
)

const (
	EncodingGzip = "gzip"
)

type Encoder interface {
	Encode(v []byte) ([]byte, error)
	Decode(v []byte) ([]byte, error)
}

func encoderFromRequest(r *Request) Encoder {
	e, ok := r.Headers.Get(HeaderAcceptEncoding)
	if !ok {
		return nil
	}
	encodings := strings.Split(e, ",")
	for _, e := range encodings {
		e = strings.TrimSpace(e)
		switch e {
		case EncodingGzip:
			return NewGzipEncoder()
		}
	}
	return nil
}

type gzipEncoder struct{}

func NewGzipEncoder() *gzipEncoder {
	return &gzipEncoder{}
}

func (e *gzipEncoder) Encode(v []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(v); err != nil {
		return nil, fmt.Errorf("failed to write: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to close: %w", err)
	}
	return buf.Bytes(), nil
}

func (e *gzipEncoder) Decode(v []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(v))
	if err != nil {
		return nil, fmt.Errorf("failed to create reader: %w", err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			log.Println("failed to close gzip reader: ", err.Error())
		}
	}()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		return nil, fmt.Errorf("failed to read: %w", err)
	}
	return buf.Bytes(), nil
}
