package main

import "strings"

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
	return v, nil
}

func (e *gzipEncoder) Decode(v []byte) ([]byte, error) {
	return v, nil
}
