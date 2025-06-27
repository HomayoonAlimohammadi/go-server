package main

import (
	"bytes"
	"context"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	dir := "/tmp/test"
	listener := &net.TCPListener{}
	shutdownCh := make(chan os.Signal, 1)

	server := NewServer(dir, listener, shutdownCh)

	if server.dir != dir {
		t.Errorf("NewServer() dir = %v, want %v", server.dir, dir)
	}

	if server.listener != listener {
		t.Errorf("NewServer() listener = %v, want %v", server.listener, listener)
	}

	if server.shutdownCh != shutdownCh {
		t.Errorf("NewServer() shutdownCh = %v, want %v", server.shutdownCh, shutdownCh)
	}

	if len(server.routes) != 0 {
		t.Errorf("NewServer() routes should be empty initially, got %d routes", len(server.routes))
	}
}

func TestServer_Register(t *testing.T) {
	server := createTestServer(t)

	// Test registering a route
	handler := func(ctx context.Context, req *Request, w io.Writer) error {
		return nil
	}

	server.Register("GET", "/test", handler)

	if len(server.routes) != 1 {
		t.Errorf("Register() routes count = %v, want 1", len(server.routes))
	}

	route := server.routes[0]
	if route.method != "GET" {
		t.Errorf("Register() method = %v, want GET", route.method)
	}

	if route.prefix != "/test" {
		t.Errorf("Register() prefix = %v, want /test", route.prefix)
	}

	// Test registering multiple routes
	server.Register("POST", "/api", handler)
	server.Register("PUT", "/data", handler)

	if len(server.routes) != 3 {
		t.Errorf("Register() after multiple registrations routes count = %v, want 3", len(server.routes))
	}
}

func TestServer_Route(t *testing.T) {
	server := createTestServer(t)

	// Register test handlers
	handlerCalled := false
	testHandler := func(ctx context.Context, req *Request, w io.Writer) error {
		handlerCalled = true
		w.Write([]byte("test response"))
		return nil
	}

	server.Register("GET", "/test", testHandler)
	server.Register("POST", "/api", testHandler)

	tests := []struct {
		name         string
		request      *Request
		wantHandler  bool
		wantNotFound bool
	}{
		{
			name:        "Match exact route",
			request:     createTestRequest("GET", "/test", "HTTP/1.1", nil, nil),
			wantHandler: true,
		},
		{
			name:        "Match route with path extension",
			request:     createTestRequest("GET", "/test/subpath", "HTTP/1.1", nil, nil),
			wantHandler: true,
		},
		{
			name:        "Match different method",
			request:     createTestRequest("POST", "/api/endpoint", "HTTP/1.1", nil, nil),
			wantHandler: true,
		},
		{
			name:         "No matching route",
			request:      createTestRequest("GET", "/nonexistent", "HTTP/1.1", nil, nil),
			wantNotFound: true,
		},
		{
			name:         "Wrong method",
			request:      createTestRequest("DELETE", "/test", "HTTP/1.1", nil, nil),
			wantNotFound: true,
		},
		{
			name:         "Similar but not matching prefix",
			request:      createTestRequest("GET", "/tes", "HTTP/1.1", nil, nil),
			wantNotFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled = false
			var buf bytes.Buffer
			ctx := context.Background()

			err := server.Route(ctx, tt.request, &buf)
			if err != nil {
				t.Errorf("Route() error = %v", err)
			}

			if tt.wantHandler && !handlerCalled {
				t.Error("Route() should have called registered handler")
			}

			if tt.wantNotFound && handlerCalled {
				t.Error("Route() should not have called handler for non-matching route")
			}

			response := buf.String()
			if tt.wantNotFound {
				// Should return 404
				if !strings.Contains(response, "404") {
					t.Error("Route() should return 404 for non-matching route")
				}
			}
		})
	}
}

func TestServer_Route_PriorityOrder(t *testing.T) {
	server := createTestServer(t)

	// Register handlers in specific order to test priority
	handler1Called := false
	handler1 := func(ctx context.Context, req *Request, w io.Writer) error {
		handler1Called = true
		w.Write([]byte("handler1"))
		return nil
	}

	handler2Called := false
	handler2 := func(ctx context.Context, req *Request, w io.Writer) error {
		handler2Called = true
		w.Write([]byte("handler2"))
		return nil
	}

	// Register more specific route first
	server.Register("GET", "/api/specific", handler1)
	// Register more general route second
	server.Register("GET", "/api", handler2)

	var buf bytes.Buffer
	ctx := context.Background()
	req := createTestRequest("GET", "/api/specific/path", "HTTP/1.1", nil, nil)

	err := server.Route(ctx, req, &buf)
	if err != nil {
		t.Errorf("Route() error = %v", err)
	}

	// First registered handler should be called (more specific)
	if !handler1Called {
		t.Error("Route() should call first matching handler")
	}

	if handler2Called {
		t.Error("Route() should not call second handler when first matches")
	}
}

func createMockTCPListener(t *testing.T) *net.TCPListener {
	// Create a listener that we can close immediately
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to resolve TCP address: %v", err)
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to create TCP listener: %v", err)
	}

	return listener
}

func TestServer_Start_ContextCancellation(t *testing.T) {
	listener := createMockTCPListener(t)
	defer listener.Close()

	shutdownCh := make(chan os.Signal, 1)
	server := NewServer("/tmp", listener, shutdownCh)

	ctx, cancel := context.WithCancel(context.Background())

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()

	// Cancel context after short delay
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Check that server stops
	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Errorf("Start() error = %v, want context.Canceled", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Start() did not stop after context cancellation")
	}
}

func TestServer_Start_ShutdownSignal(t *testing.T) {
	listener := createMockTCPListener(t)
	defer listener.Close()

	shutdownCh := make(chan os.Signal, 1)
	server := NewServer("/tmp", listener, shutdownCh)

	ctx := context.Background()

	// Start server in goroutine
	go func() {
		server.Start(ctx)
	}()

	// Send shutdown signal after short delay
	time.Sleep(10 * time.Millisecond)
	shutdownCh <- os.Interrupt

	// Give some time for shutdown processing
	time.Sleep(50 * time.Millisecond)

	// Test passes if no deadlock occurs
}

// Test route matching edge cases
func TestServer_Route_EdgeCases(t *testing.T) {
	server := createTestServer(t)

	handlerCalled := false
	testHandler := func(ctx context.Context, req *Request, w io.Writer) error {
		handlerCalled = true
		return nil
	}

	// Register routes with edge cases
	server.Register("GET", "/", testHandler)             // Root path
	server.Register("GET", "/a", testHandler)            // Single character
	server.Register("GET", "/api/v1/users", testHandler) // Nested path

	tests := []struct {
		name        string
		request     *Request
		shouldMatch bool
	}{
		{
			name:        "Root path exact match",
			request:     createTestRequest("GET", "/", "HTTP/1.1", nil, nil),
			shouldMatch: true,
		},
		{
			name:        "Root path with extension",
			request:     createTestRequest("GET", "/anything", "HTTP/1.1", nil, nil),
			shouldMatch: true, // "/" prefix matches anything
		},
		{
			name:        "Single character exact",
			request:     createTestRequest("GET", "/a", "HTTP/1.1", nil, nil),
			shouldMatch: true,
		},
		{
			name:        "Single character with extension",
			request:     createTestRequest("GET", "/a/b/c", "HTTP/1.1", nil, nil),
			shouldMatch: true,
		},
		{
			name:        "Nested path exact",
			request:     createTestRequest("GET", "/api/v1/users", "HTTP/1.1", nil, nil),
			shouldMatch: true,
		},
		{
			name:        "Nested path with extension",
			request:     createTestRequest("GET", "/api/v1/users/123", "HTTP/1.1", nil, nil),
			shouldMatch: true,
		},
		{
			name:        "Case sensitive path",
			request:     createTestRequest("GET", "/API/v1/users", "HTTP/1.1", nil, nil),
			shouldMatch: true, // Root path "/" will match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled = false
			var buf bytes.Buffer
			ctx := context.Background()

			err := server.Route(ctx, tt.request, &buf)
			if err != nil {
				t.Errorf("Route() error = %v", err)
			}

			if tt.shouldMatch && !handlerCalled {
				t.Error("Route() should have matched and called handler")
			}

			response := buf.String()
			if tt.shouldMatch && strings.Contains(response, "404") {
				t.Error("Route() should not return 404 for matching route")
			}
		})
	}
}

// Benchmark tests for server operations
func BenchmarkServer_Route(b *testing.B) {
	server := createTestServer(&testing.T{})

	handler := func(ctx context.Context, req *Request, w io.Writer) error {
		return httpResponse(w, 200, nil, "OK")
	}

	// Register multiple routes
	server.Register("GET", "/", handler)
	server.Register("GET", "/api", handler)
	server.Register("GET", "/api/users", handler)
	server.Register("POST", "/api/users", handler)
	server.Register("GET", "/files", handler)

	request := createTestRequest("GET", "/api/users/123", "HTTP/1.1", nil, nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		server.Route(ctx, request, &buf)
	}
}

func BenchmarkServer_Register(b *testing.B) {
	server := createTestServer(&testing.T{})

	handler := func(ctx context.Context, req *Request, w io.Writer) error {
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.Register("GET", "/test", handler)
	}
}
