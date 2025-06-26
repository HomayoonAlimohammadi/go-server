package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type match struct {
	method  string
	prefix  string
	handler handleFunc
}
type server struct {
	dir        string
	routes     []match
	listener   *net.TCPListener
	shutdownCh <-chan os.Signal
}

func NewServer(dir string, listener *net.TCPListener, shutdownCh <-chan os.Signal) *server {
	s := &server{
		dir:        dir,
		listener:   listener,
		shutdownCh: shutdownCh,
	}
	return s
}

func (s *server) Register(method string, prefix string, handler handleFunc) {
	s.routes = append(s.routes, match{method: method, prefix: prefix, handler: handler})
}

func (s *server) Route(ctx context.Context, req *Request, w io.Writer) error {
	for _, m := range s.routes {
		if req.Method == m.method && strings.HasPrefix(req.Target, m.prefix) {
			log.Printf("Matched method=%s prefix=%s", m.method, m.prefix)
			return m.handler(ctx, req, w)
		}
	}
	log.Println("Did not match any route")
	return s.handleNotFound(ctx, req, w)
}

func (s *server) Start(ctx context.Context) error {
	// Set a short read timeout on the listener to make it non-blocking
	go func(ctx context.Context) {
		select {
		case <-ctx.Done():
		case <-s.shutdownCh:
		}
		log.Println("Server is shutting down...")
	}(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Set a short deadline to make Accept non-blocking
		s.listener.SetDeadline(time.Now().Add(1 * time.Second))

		conn, err := s.listener.Accept()
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				// Timeout occurred, continue to check context
				continue
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				log.Println("Error accepting connection: ", err.Error())
				continue
			}
		}

		go func(conn net.Conn) {
			if err := s.handleConn(conn); err != nil {
				log.Println("Error handling connection: ", err.Error())
			}
		}(conn)
	}

}

func (s *server) handleConn(conn net.Conn) error {
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	d, _ := ctx.Deadline()

	if err := conn.SetDeadline(d); err != nil {
		log.Println("Error setting deadline: ", err.Error())
	}

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return fmt.Errorf("failed to read conn: %w", err)
	}

	b := buf[:n]
	req := &Request{}
	if err := req.From(b); err != nil {
		return fmt.Errorf("failed to read request: %w", err)
	}

	log.Printf("Request: %+v", req)

	if err := s.Route(ctx, req, conn); err != nil {
		return fmt.Errorf("failed to handle request: %w", err)
	}

	return nil
}
