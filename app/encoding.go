package main

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
	switch e {
	case EncodingGzip:
		return NewGzipEncoder()
	default:
		return nil
	}
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
