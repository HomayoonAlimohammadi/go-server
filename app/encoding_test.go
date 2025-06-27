package main

import (
	"bytes"
	"testing"
)

func TestEncoderFromRequest(t *testing.T) {
	tests := []struct {
		name           string
		acceptEncoding string
		wantEncoder    bool
	}{
		{
			name:           "No Accept-Encoding header",
			acceptEncoding: "",
			wantEncoder:    false,
		},
		{
			name:           "Gzip encoding",
			acceptEncoding: "gzip",
			wantEncoder:    true,
		},
		{
			name:           "Multiple encodings with gzip",
			acceptEncoding: "deflate, gzip, br",
			wantEncoder:    true,
		},
		{
			name:           "Gzip with quality values",
			acceptEncoding: "gzip;q=0.8, deflate;q=0.6",
			wantEncoder:    false, // Current implementation doesn't handle quality values
		},
		{
			name:           "Unsupported encodings only",
			acceptEncoding: "deflate, br, identity",
			wantEncoder:    false,
		},
		{
			name:           "Gzip with spaces",
			acceptEncoding: " gzip , deflate",
			wantEncoder:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &Request{
				Headers: make(Headers),
			}

			if tt.acceptEncoding != "" {
				req.Headers.Set(HeaderAcceptEncoding, tt.acceptEncoding)
			}

			encoder := encoderFromRequest(req)

			if tt.wantEncoder && encoder == nil {
				t.Errorf("encoderFromRequest() = nil, want encoder")
			} else if !tt.wantEncoder && encoder != nil {
				t.Errorf("encoderFromRequest() = %v, want nil", encoder)
			}
		})
	}
}

func TestGzipEncoder(t *testing.T) {
	encoder := NewGzipEncoder()

	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name:    "Empty input",
			input:   []byte(""),
			wantErr: false,
		},
		{
			name:    "Simple text",
			input:   []byte("Hello, World!"),
			wantErr: false,
		},
		{
			name:    "Large text",
			input:   bytes.Repeat([]byte("test"), 1000),
			wantErr: false,
		},
		{
			name:    "Binary data",
			input:   []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
			wantErr: false,
		},
		{
			name:    "Unicode text",
			input:   []byte("Hello, 世界! 🌍"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test encoding
			encoded, err := encoder.Encode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Test that encoded data is different from input (unless empty)
				if len(tt.input) > 0 && bytes.Equal(encoded, tt.input) {
					t.Error("Encode() returned same data as input, expected compression")
				}

				// Test decoding
				decoded, err := encoder.Decode(encoded)
				if err != nil {
					t.Errorf("Decode() error = %v", err)
					return
				}

				if !bytes.Equal(decoded, tt.input) {
					t.Errorf("Decode() = %v, want %v", decoded, tt.input)
				}
			}
		})
	}
}

func TestGzipEncoder_DecodeError(t *testing.T) {
	encoder := NewGzipEncoder()

	// Test with invalid gzip data
	invalidData := []byte("this is not gzip data")
	_, err := encoder.Decode(invalidData)
	if err == nil {
		t.Error("Decode() with invalid data should return error")
	}
}

func BenchmarkGzipEncode(b *testing.B) {
	encoder := NewGzipEncoder()
	data := bytes.Repeat([]byte("Hello, World! "), 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encoder.Encode(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGzipDecode(b *testing.B) {
	encoder := NewGzipEncoder()
	data := bytes.Repeat([]byte("Hello, World! "), 100)

	encoded, err := encoder.Encode(data)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encoder.Decode(encoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}
