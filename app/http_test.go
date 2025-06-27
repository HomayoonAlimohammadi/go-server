package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestHeaders_Get(t *testing.T) {
	headers := make(Headers)
	headers["Content-Type"] = "text/plain"
	headers["content-length"] = "123"
	headers["Connection"] = "keep-alive"

	tests := []struct {
		name      string
		key       string
		wantValue string
		wantFound bool
	}{
		{
			name:      "Exact case match",
			key:       "Content-Type",
			wantValue: "text/plain",
			wantFound: true,
		},
		{
			name:      "Case insensitive match",
			key:       "content-type",
			wantValue: "text/plain",
			wantFound: true,
		},
		{
			name:      "Mixed case match",
			key:       "CoNtEnT-tYpE",
			wantValue: "text/plain",
			wantFound: true,
		},
		{
			name:      "Different case header",
			key:       "CONTENT-LENGTH",
			wantValue: "123",
			wantFound: true,
		},
		{
			name:      "Non-existent header",
			key:       "Authorization",
			wantValue: "",
			wantFound: false,
		},
		{
			name:      "Empty key",
			key:       "",
			wantValue: "",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, found := headers.Get(tt.key)
			if found != tt.wantFound {
				t.Errorf("Headers.Get() found = %v, want %v", found, tt.wantFound)
			}
			if value != tt.wantValue {
				t.Errorf("Headers.Get() value = %v, want %v", value, tt.wantValue)
			}
		})
	}
}

func TestHeaders_Set(t *testing.T) {
	tests := []struct {
		name         string
		initialKey   string
		initialValue string
		setKey       string
		setValue     string
		wantKey      string
		wantValue    string
	}{
		{
			name:      "Set new header",
			setKey:    "Content-Type",
			setValue:  "text/plain",
			wantKey:   "Content-Type",
			wantValue: "text/plain",
		},
		{
			name:         "Update existing header same case",
			initialKey:   "Content-Type",
			initialValue: "text/html",
			setKey:       "Content-Type",
			setValue:     "text/plain",
			wantKey:      "Content-Type",
			wantValue:    "text/plain",
		},
		{
			name:         "Update existing header different case",
			initialKey:   "Content-Type",
			initialValue: "text/html",
			setKey:       "content-type",
			setValue:     "text/plain",
			wantKey:      "Content-Type",
			wantValue:    "text/plain",
		},
		{
			name:      "Set header with spaces",
			setKey:    "  Content-Type  ",
			setValue:  "text/plain",
			wantKey:   "Content-Type",
			wantValue: "text/plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := make(Headers)

			if tt.initialKey != "" {
				headers[tt.initialKey] = tt.initialValue
			}

			headers.Set(tt.setKey, tt.setValue)

			// Check that the value is set correctly
			value, found := headers.Get(tt.wantKey)
			if !found {
				t.Errorf("Headers.Set() header not found after setting")
			}
			if value != tt.wantValue {
				t.Errorf("Headers.Set() value = %v, want %v", value, tt.wantValue)
			}

			// Check that only one entry exists for the header
			count := 0
			for key := range headers {
				if strings.EqualFold(key, tt.wantKey) {
					count++
				}
			}
			if count != 1 {
				t.Errorf("Headers.Set() created %d entries, want 1", count)
			}
		})
	}
}

func TestRequest_From(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantMethod  string
		wantTarget  string
		wantVersion string
		wantHeaders map[string]string
		wantBody    string
		wantErr     bool
	}{
		{
			name:        "Basic GET request",
			input:       "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n",
			wantMethod:  "GET",
			wantTarget:  "/",
			wantVersion: "HTTP/1.1",
			wantHeaders: map[string]string{
				"Host": "localhost",
			},
			wantBody: "",
		},
		{
			name:        "POST request with body",
			input:       "POST /api/data HTTP/1.1\r\nContent-Type: application/json\r\nContent-Length: 13\r\n\r\n{\"test\":true}",
			wantMethod:  "POST",
			wantTarget:  "/api/data",
			wantVersion: "HTTP/1.1",
			wantHeaders: map[string]string{
				"Content-Type":   "application/json",
				"Content-Length": "13",
			},
			wantBody: "{\"test\":true}",
		},
		{
			name:        "Request with multiple headers",
			input:       "GET /echo/hello HTTP/1.1\r\nHost: localhost:4221\r\nUser-Agent: curl/7.68.0\r\nAccept: */*\r\nConnection: keep-alive\r\n\r\n",
			wantMethod:  "GET",
			wantTarget:  "/echo/hello",
			wantVersion: "HTTP/1.1",
			wantHeaders: map[string]string{
				"Host":       "localhost:4221",
				"User-Agent": "curl/7.68.0",
				"Accept":     "*/*",
				"Connection": "keep-alive",
			},
			wantBody: "",
		},
		{
			name:        "Request with header containing colon",
			input:       "GET / HTTP/1.1\r\nAuthorization: Bearer token:with:colons\r\n\r\n",
			wantMethod:  "GET",
			wantTarget:  "/",
			wantVersion: "HTTP/1.1",
			wantHeaders: map[string]string{
				"Authorization": "Bearer token:with:colons",
			},
			wantBody: "",
		},
		{
			name:        "Request with empty body",
			input:       "PUT /data HTTP/1.1\r\nContent-Length: 0\r\n\r\n",
			wantMethod:  "PUT",
			wantTarget:  "/data",
			wantVersion: "HTTP/1.1",
			wantHeaders: map[string]string{
				"Content-Length": "0",
			},
			wantBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &Request{}
			err := req.From([]byte(tt.input))

			if (err != nil) != tt.wantErr {
				t.Errorf("Request.From() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if req.Method != tt.wantMethod {
				t.Errorf("Request.From() Method = %v, want %v", req.Method, tt.wantMethod)
			}

			if req.Target != tt.wantTarget {
				t.Errorf("Request.From() Target = %v, want %v", req.Target, tt.wantTarget)
			}

			if req.Version != tt.wantVersion {
				t.Errorf("Request.From() Version = %v, want %v", req.Version, tt.wantVersion)
			}

			if string(req.Body) != tt.wantBody {
				t.Errorf("Request.From() Body = %v, want %v", string(req.Body), tt.wantBody)
			}

			for key, expectedValue := range tt.wantHeaders {
				if actualValue, found := req.Headers.Get(key); !found {
					t.Errorf("Request.From() missing header %s", key)
				} else if actualValue != expectedValue {
					t.Errorf("Request.From() header %s = %v, want %v", key, actualValue, expectedValue)
				}
			}
		})
	}
}

func TestRequest_String(t *testing.T) {
	req := &Request{
		Method:  "GET",
		Target:  "/test",
		Version: "HTTP/1.1",
		Headers: make(Headers),
		Body:    []byte("test body"),
	}
	req.Headers.Set("Content-Type", "text/plain")
	req.Headers.Set("Content-Length", "9")

	result := req.String()

	// Check that the string contains expected parts
	expectedParts := []string{
		"GET",
		"/test",
		"HTTP/1.1",
		"Content-Type",
		"text/plain",
		"test body",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Request.String() missing expected part: %s", part)
		}
	}
}

func TestHttpResponse(t *testing.T) {
	tests := []struct {
		name         string
		code         int
		headers      Headers
		body         interface{}
		wantContains []string
		wantStatus   string
	}{
		{
			name:       "Basic 200 response",
			code:       200,
			headers:    nil,
			body:       "Hello",
			wantStatus: "HTTP/1.1 200 OK",
			wantContains: []string{
				"Content-Length: 5",
				"Connection: keep-alive",
				"Hello",
			},
		},
		{
			name:       "404 response",
			code:       404,
			headers:    nil,
			body:       "",
			wantStatus: "HTTP/1.1 404 Not Found",
			wantContains: []string{
				"Connection: keep-alive",
			},
		},
		{
			name: "Response with custom headers",
			code: 200,
			headers: func() Headers {
				h := make(Headers)
				h.Set("Content-Type", "application/json")
				h.Set("Cache-Control", "no-cache")
				return h
			}(),
			body:       `{"status":"ok"}`,
			wantStatus: "HTTP/1.1 200 OK",
			wantContains: []string{
				"Content-Type: application/json",
				"Cache-Control: no-cache",
				"Content-Length: 15",
				`{"status":"ok"}`,
			},
		},
		{
			name: "Response with connection close",
			code: 200,
			headers: func() Headers {
				h := make(Headers)
				h.Set("Connection", "close")
				return h
			}(),
			body:       "closing",
			wantStatus: "HTTP/1.1 200 OK",
			wantContains: []string{
				"Connection: close",
				"Content-Length: 7",
				"closing",
			},
		},
		{
			name:       "Empty body response",
			code:       201,
			headers:    nil,
			body:       "",
			wantStatus: "HTTP/1.1 201 Created",
			wantContains: []string{
				"Connection: keep-alive",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := httpResponse(&buf, tt.code, tt.headers, tt.body)
			if err != nil {
				t.Errorf("httpResponse() error = %v", err)
				return
			}

			response := buf.String()

			// Check status line
			if !strings.HasPrefix(response, tt.wantStatus) {
				t.Errorf("httpResponse() status = %v, want prefix %v", response, tt.wantStatus)
			}

			// Check for expected content
			for _, content := range tt.wantContains {
				if !strings.Contains(response, content) {
					t.Errorf("httpResponse() missing expected content: %s\nFull response:\n%s", content, response)
				}
			}

			// Check that response ends with double CRLF if there's a body
			if tt.body != "" && !strings.Contains(response, "\r\n\r\n") {
				t.Error("httpResponse() missing double CRLF separator")
			}
		})
	}
}

func BenchmarkHeaders_Get(b *testing.B) {
	headers := make(Headers)
	headers["Content-Type"] = "text/plain"
	headers["Content-Length"] = "100"
	headers["Connection"] = "keep-alive"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		headers.Get("content-type")
	}
}

func BenchmarkHeaders_Set(b *testing.B) {
	headers := make(Headers)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		headers.Set("Content-Type", "text/plain")
	}
}

func BenchmarkRequest_From(b *testing.B) {
	requestData := []byte("GET /test HTTP/1.1\r\nHost: localhost\r\nUser-Agent: test\r\n\r\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &Request{}
		req.From(requestData)
	}
}

func BenchmarkHttpResponse(b *testing.B) {
	headers := make(Headers)
	headers.Set("Content-Type", "text/plain")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		httpResponse(&buf, 200, headers, "Hello, World!")
	}
}
