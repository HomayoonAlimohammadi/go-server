package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper function to create a test server
func createTestServer(t *testing.T) *server {
	tempDir := t.TempDir()
	return &server{
		dir: tempDir,
	}
}

// Helper function to create a test request
func createTestRequest(method, target, version string, headers map[string]string, body []byte) *Request {
	req := &Request{
		Method:  method,
		Target:  target,
		Version: version,
		Headers: make(Headers),
		Body:    body,
	}

	for k, v := range headers {
		req.Headers.Set(k, v)
	}

	return req
}

// Helper function to capture response
func captureResponse(t *testing.T, handler func(context.Context, *Request, io.Writer) error, req *Request) *bytes.Buffer {
	var buf bytes.Buffer
	ctx := context.Background()
	err := handler(ctx, req, &buf)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	return &buf
}

// Helper function to parse HTTP response
func parseHTTPResponse(response string) (statusCode int, headers map[string]string, body string) {
	lines := strings.Split(response, "\r\n")

	// Parse status line
	statusLine := strings.Split(lines[0], " ")
	if len(statusLine) >= 2 {
		switch statusLine[1] {
		case "200":
			statusCode = 200
		case "201":
			statusCode = 201
		case "404":
			statusCode = 404
		case "500":
			statusCode = 500
		}
	}

	// Parse headers
	headers = make(map[string]string)
	bodyStart := 0
	for i := 1; i < len(lines); i++ {
		if lines[i] == "" {
			bodyStart = i + 1
			break
		}
		if strings.Contains(lines[i], ":") {
			parts := strings.SplitN(lines[i], ":", 2)
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	// Parse body
	if bodyStart < len(lines) {
		body = strings.Join(lines[bodyStart:], "\r\n")
	}

	return
}

func TestRootGet(t *testing.T) {
	server := createTestServer(t)

	tests := []struct {
		name     string
		request  *Request
		wantCode int
		wantBody string
	}{
		{
			name:     "Basic root request",
			request:  createTestRequest("GET", "/", "HTTP/1.1", nil, nil),
			wantCode: 200,
			wantBody: "",
		},
		{
			name: "Root request with headers",
			request: createTestRequest("GET", "/", "HTTP/1.1", map[string]string{
				"Connection": "keep-alive",
				"Host":       "localhost",
			}, nil),
			wantCode: 200,
			wantBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := captureResponse(t, server.rootGet, tt.request)
			statusCode, headers, body := parseHTTPResponse(buf.String())

			if statusCode != tt.wantCode {
				t.Errorf("rootGet() status = %v, want %v", statusCode, tt.wantCode)
			}

			if body != tt.wantBody {
				t.Errorf("rootGet() body = %v, want %v", body, tt.wantBody)
			}

			// Check that Connection header is set
			if _, exists := headers["Connection"]; !exists {
				t.Error("rootGet() missing Connection header")
			}
		})
	}
}

func TestHandleNotFound(t *testing.T) {
	server := createTestServer(t)

	tests := []struct {
		name     string
		request  *Request
		wantCode int
		wantBody string
	}{
		{
			name:     "Basic not found request",
			request:  createTestRequest("GET", "/nonexistent", "HTTP/1.1", nil, nil),
			wantCode: 404,
			wantBody: "",
		},
		{
			name: "Not found with headers",
			request: createTestRequest("POST", "/unknown", "HTTP/1.1", map[string]string{
				"Connection": "close",
			}, nil),
			wantCode: 404,
			wantBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := captureResponse(t, server.handleNotFound, tt.request)
			statusCode, _, body := parseHTTPResponse(buf.String())

			if statusCode != tt.wantCode {
				t.Errorf("handleNotFound() status = %v, want %v", statusCode, tt.wantCode)
			}

			if body != tt.wantBody {
				t.Errorf("handleNotFound() body = %v, want %v", body, tt.wantBody)
			}
		})
	}
}

func TestEchoGet(t *testing.T) {
	server := createTestServer(t)

	tests := []struct {
		name        string
		request     *Request
		wantCode    int
		wantBody    string
		wantHeaders map[string]string
	}{
		{
			name:     "Basic echo request",
			request:  createTestRequest("GET", "/echo/hello", "HTTP/1.1", nil, nil),
			wantCode: 200,
			wantBody: "hello",
			wantHeaders: map[string]string{
				"Content-Type": "text/plain",
			},
		},
		{
			name:     "Echo with spaces",
			request:  createTestRequest("GET", "/echo/hello world", "HTTP/1.1", nil, nil),
			wantCode: 200,
			wantBody: "hello world",
		},
		{
			name:     "Echo empty string",
			request:  createTestRequest("GET", "/echo/", "HTTP/1.1", nil, nil),
			wantCode: 200,
			wantBody: "",
		},
		{
			name: "Echo with gzip encoding",
			request: createTestRequest("GET", "/echo/compress", "HTTP/1.1", map[string]string{
				"Accept-Encoding": "gzip",
			}, nil),
			wantCode: 200,
			wantHeaders: map[string]string{
				"Content-Type":     "text/plain",
				"Content-Encoding": "gzip",
			},
		},
		{
			name: "Echo with multiple encodings",
			request: createTestRequest("GET", "/echo/test", "HTTP/1.1", map[string]string{
				"Accept-Encoding": "deflate, gzip, br",
			}, nil),
			wantCode: 200,
			wantHeaders: map[string]string{
				"Content-Type":     "text/plain",
				"Content-Encoding": "gzip",
			},
		},
		{
			name: "Echo with unsupported encoding",
			request: createTestRequest("GET", "/echo/test", "HTTP/1.1", map[string]string{
				"Accept-Encoding": "deflate, br",
			}, nil),
			wantCode: 200,
			wantBody: "test",
			wantHeaders: map[string]string{
				"Content-Type": "text/plain",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := captureResponse(t, server.echoGet, tt.request)
			statusCode, headers, body := parseHTTPResponse(buf.String())

			if statusCode != tt.wantCode {
				t.Errorf("echoGet() status = %v, want %v", statusCode, tt.wantCode)
			}

			// For gzip encoded responses, we can't easily check the body content
			// but we can verify the headers are correct
			if tt.wantHeaders != nil {
				for key, expectedValue := range tt.wantHeaders {
					if actualValue, exists := headers[key]; !exists {
						t.Errorf("echoGet() missing header %s", key)
					} else if actualValue != expectedValue {
						t.Errorf("echoGet() header %s = %v, want %v", key, actualValue, expectedValue)
					}
				}
			}

			// Only check body for non-gzip responses
			if _, hasGzip := headers["Content-Encoding"]; !hasGzip && tt.wantBody != "" {
				if body != tt.wantBody {
					t.Errorf("echoGet() body = %v, want %v", body, tt.wantBody)
				}
			}
		})
	}
}

func TestUserAgentGet(t *testing.T) {
	server := createTestServer(t)

	tests := []struct {
		name     string
		request  *Request
		wantCode int
		wantBody string
	}{
		{
			name: "Basic user agent request",
			request: createTestRequest("GET", "/user-agent", "HTTP/1.1", map[string]string{
				"User-Agent": "Mozilla/5.0",
			}, nil),
			wantCode: 200,
			wantBody: "Mozilla/5.0",
		},
		{
			name: "Complex user agent",
			request: createTestRequest("GET", "/user-agent", "HTTP/1.1", map[string]string{
				"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			}, nil),
			wantCode: 200,
			wantBody: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		},
		{
			name:     "Missing user agent",
			request:  createTestRequest("GET", "/user-agent", "HTTP/1.1", nil, nil),
			wantCode: 200,
			wantBody: "",
		},
		{
			name: "Empty user agent",
			request: createTestRequest("GET", "/user-agent", "HTTP/1.1", map[string]string{
				"User-Agent": "",
			}, nil),
			wantCode: 200,
			wantBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := captureResponse(t, server.userAgentGet, tt.request)
			statusCode, headers, body := parseHTTPResponse(buf.String())

			if statusCode != tt.wantCode {
				t.Errorf("userAgentGet() status = %v, want %v", statusCode, tt.wantCode)
			}

			if body != tt.wantBody {
				t.Errorf("userAgentGet() body = %v, want %v", body, tt.wantBody)
			}

			// Check content type
			if contentType, exists := headers["Content-Type"]; !exists {
				t.Error("userAgentGet() missing Content-Type header")
			} else if contentType != "text/plain" {
				t.Errorf("userAgentGet() Content-Type = %v, want text/plain", contentType)
			}
		})
	}
}

func TestFilesGet(t *testing.T) {
	server := createTestServer(t)

	// Create test files
	testFile := filepath.Join(server.dir, "test.txt")
	testContent := "Hello, World!"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	binaryFile := filepath.Join(server.dir, "binary.bin")
	binaryContent := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}
	if err := os.WriteFile(binaryFile, binaryContent, 0644); err != nil {
		t.Fatalf("Failed to create binary test file: %v", err)
	}

	tests := []struct {
		name        string
		request     *Request
		wantCode    int
		wantBody    string
		wantHeaders map[string]string
	}{
		{
			name:     "Get existing file",
			request:  createTestRequest("GET", "/files/test.txt", "HTTP/1.1", nil, nil),
			wantCode: 200,
			wantBody: testContent,
			wantHeaders: map[string]string{
				"Content-Type": "application/octet-stream",
			},
		},
		{
			name:     "Get non-existent file",
			request:  createTestRequest("GET", "/files/nonexistent.txt", "HTTP/1.1", nil, nil),
			wantCode: 404,
			wantBody: "",
		},
		{
			name:     "Get binary file",
			request:  createTestRequest("GET", "/files/binary.bin", "HTTP/1.1", nil, nil),
			wantCode: 200,
			wantBody: string(binaryContent),
			wantHeaders: map[string]string{
				"Content-Type": "application/octet-stream",
			},
		},
		{
			name:     "Get file with path traversal attempt",
			request:  createTestRequest("GET", "/files/../etc/passwd", "HTTP/1.1", nil, nil),
			wantCode: 404,
			wantBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := captureResponse(t, server.filesGet, tt.request)
			statusCode, headers, body := parseHTTPResponse(buf.String())

			if statusCode != tt.wantCode {
				t.Errorf("filesGet() status = %v, want %v", statusCode, tt.wantCode)
			}

			if tt.wantHeaders != nil {
				for key, expectedValue := range tt.wantHeaders {
					if actualValue, exists := headers[key]; !exists {
						t.Errorf("filesGet() missing header %s", key)
					} else if actualValue != expectedValue {
						t.Errorf("filesGet() header %s = %v, want %v", key, actualValue, expectedValue)
					}
				}
			}

			if tt.wantBody != "" && body != tt.wantBody {
				t.Errorf("filesGet() body = %v, want %v", body, tt.wantBody)
			}
		})
	}
}

// Test file read error by creating a file without read permissions
func TestFilesGet_ReadError(t *testing.T) {
	server := createTestServer(t)

	// Create a file and remove read permissions
	testFile := filepath.Join(server.dir, "noread.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Remove read permissions
	if err := os.Chmod(testFile, 0000); err != nil {
		t.Fatalf("Failed to change file permissions: %v", err)
	}

	// Restore permissions after test
	defer func() {
		os.Chmod(testFile, 0644)
	}()

	request := createTestRequest("GET", "/files/noread.txt", "HTTP/1.1", nil, nil)
	buf := captureResponse(t, server.filesGet, request)
	statusCode, headers, _ := parseHTTPResponse(buf.String())

	if statusCode != 500 {
		t.Errorf("filesGet() with read error status = %v, want 500", statusCode)
	}

	if contentType, exists := headers["Content-Type"]; !exists {
		t.Error("filesGet() with read error missing Content-Type header")
	} else if contentType != "text/plain" {
		t.Errorf("filesGet() with read error Content-Type = %v, want text/plain", contentType)
	}
}

func TestFilesPost(t *testing.T) {
	server := createTestServer(t)

	tests := []struct {
		name            string
		request         *Request
		wantCode        int
		checkFile       bool
		expectedContent string
	}{
		{
			name:            "Create new file",
			request:         createTestRequest("POST", "/files/new.txt", "HTTP/1.1", nil, []byte("Hello, POST!")),
			wantCode:        201,
			checkFile:       true,
			expectedContent: "Hello, POST!",
		},
		{
			name:            "Overwrite existing file",
			request:         createTestRequest("POST", "/files/existing.txt", "HTTP/1.1", nil, []byte("Updated content")),
			wantCode:        201,
			checkFile:       true,
			expectedContent: "Updated content",
		},
		{
			name:            "Create file with empty content",
			request:         createTestRequest("POST", "/files/empty.txt", "HTTP/1.1", nil, []byte("")),
			wantCode:        201,
			checkFile:       true,
			expectedContent: "",
		},
		{
			name:            "Create binary file",
			request:         createTestRequest("POST", "/files/binary.bin", "HTTP/1.1", nil, []byte{0x00, 0x01, 0xFF}),
			wantCode:        201,
			checkFile:       true,
			expectedContent: string([]byte{0x00, 0x01, 0xFF}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := captureResponse(t, server.filesPost, tt.request)
			statusCode, _, body := parseHTTPResponse(buf.String())

			if statusCode != tt.wantCode {
				t.Errorf("filesPost() status = %v, want %v", statusCode, tt.wantCode)
			}

			if body != "" {
				t.Errorf("filesPost() body = %v, want empty", body)
			}

			if tt.checkFile {
				fileName := strings.TrimPrefix(tt.request.Target, "/files/")
				filePath := filepath.Join(server.dir, fileName)

				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Errorf("filesPost() failed to read created file: %v", err)
				} else if string(content) != tt.expectedContent {
					t.Errorf("filesPost() file content = %v, want %v", string(content), tt.expectedContent)
				}
			}
		})
	}
}

// Test file write error by trying to write to a read-only directory
func TestFilesPost_WriteError(t *testing.T) {
	server := createTestServer(t)

	// Create a subdirectory and make it read-only
	subDir := filepath.Join(server.dir, "readonly")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	if err := os.Chmod(subDir, 0444); err != nil {
		t.Fatalf("Failed to change directory permissions: %v", err)
	}

	// Restore permissions after test
	defer func() {
		os.Chmod(subDir, 0755)
	}()

	request := createTestRequest("POST", "/files/readonly/test.txt", "HTTP/1.1", nil, []byte("test"))
	buf := captureResponse(t, server.filesPost, request)
	statusCode, headers, _ := parseHTTPResponse(buf.String())

	if statusCode != 500 {
		t.Errorf("filesPost() with write error status = %v, want 500", statusCode)
	}

	if contentType, exists := headers["Content-Type"]; !exists {
		t.Error("filesPost() with write error missing Content-Type header")
	} else if contentType != "text/plain" {
		t.Errorf("filesPost() with write error Content-Type = %v, want text/plain", contentType)
	}
}

// Test helper functions
func TestNewResponseHeaders(t *testing.T) {
	reqHeaders := make(Headers)
	reqHeaders.Set("Connection", "keep-alive")
	reqHeaders.Set("Host", "localhost")
	reqHeaders.Set("User-Agent", "test")

	respHeaders := NewResponseHeaders(reqHeaders)

	// Should only copy Connection header
	if connection, exists := respHeaders.Get("Connection"); !exists {
		t.Error("NewResponseHeaders() missing Connection header")
	} else if connection != "keep-alive" {
		t.Errorf("NewResponseHeaders() Connection = %v, want keep-alive", connection)
	}

	// Should not copy other headers
	if _, exists := respHeaders.Get("Host"); exists {
		t.Error("NewResponseHeaders() should not copy Host header")
	}

	if _, exists := respHeaders.Get("User-Agent"); exists {
		t.Error("NewResponseHeaders() should not copy User-Agent header")
	}
}

// Benchmark tests
func BenchmarkRootGet(b *testing.B) {
	server := createTestServer(&testing.T{})
	request := createTestRequest("GET", "/", "HTTP/1.1", nil, nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		server.rootGet(ctx, request, &buf)
	}
}

func BenchmarkEchoGet(b *testing.B) {
	server := createTestServer(&testing.T{})
	request := createTestRequest("GET", "/echo/benchmark", "HTTP/1.1", nil, nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		server.echoGet(ctx, request, &buf)
	}
}

func BenchmarkUserAgentGet(b *testing.B) {
	server := createTestServer(&testing.T{})
	request := createTestRequest("GET", "/user-agent", "HTTP/1.1", map[string]string{
		"User-Agent": "Mozilla/5.0",
	}, nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		server.userAgentGet(ctx, request, &buf)
	}
}
