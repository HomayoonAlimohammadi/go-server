package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		log.Println("Failed to bind to port 4221: ", err.Error())
	}

	r := NewRouter()
	r.register(http.MethodGet, "/user-agent", handleUserAgent)
	r.register(http.MethodGet, "/echo", handleEcho)
	r.register(http.MethodGet, "/", handleRoot)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("Error accepting connection: ", err.Error())
		}

		go func(conn net.Conn) {
			defer conn.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			d, _ := ctx.Deadline()

			if err := conn.SetDeadline(d); err != nil {
				log.Println("Error setting deadline: ", err.Error())
			}

			if err := handleConn(ctx, conn, r); err != nil {
				log.Println("Error handling connection: ", err.Error())
			}
		}(conn)

	}
}

func handleConn(ctx context.Context, conn net.Conn, r *router) error {
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

	if err := r.Route(ctx, req, conn); err != nil {
		return fmt.Errorf("failed to handle request: %w", err)
	}

	return nil
}
