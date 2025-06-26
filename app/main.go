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

// Ensures gofmt doesn't remove the "net" and "os" imports above (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

const (
	CRLF = "\r\n"
)

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	// Uncomment this block to pass the first stage
	//
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		log.Println("Failed to bind to port 4221: ", err.Error())
	}

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

			if err := handleConn(ctx, conn); err != nil {
				log.Println("Error handling connection: ", err.Error())
			}
		}(conn)

	}
}

type Request struct {
	Method  string
	Target  string
	Version string
	Headers map[string]string
	Body    []byte
}

func (r *Request) From(b []byte) error {
	s := string(b)
	parts := strings.Split(s, "\r\n")

	// Meta
	metaSegs := strings.Split(parts[0], " ")

	r.Method = strings.TrimSpace(metaSegs[0])
	r.Target = strings.TrimSpace(metaSegs[1])
	r.Version = strings.TrimSpace(metaSegs[2])

	// Headers
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	for _, l := range parts[1 : len(parts)-1] {
		if len(l) == 0 {
			continue
		}
		kv := strings.SplitN(l, ":", 2)
		k, v := kv[0], kv[1]
		r.Headers[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}

	// Body
	r.Body = []byte(parts[len(parts)-1])

	return nil
}

func (r *Request) String() string {
	s := fmt.Sprintf("%s\n%s\n%s\n%s", r.Method, r.Target, r.Version, CRLF)
	for k, v := range r.Headers {
		s += fmt.Sprintf("%s:	%s\n", k, v)
	}
	s += fmt.Sprintf("%s%s", CRLF, r.Body)
	return s
}

func handleConn(ctx context.Context, conn net.Conn) error {
	var buffer []byte
	temp := make([]byte, 1024)

	for {
		n, err := conn.Read(temp)
		if err != nil {
			return fmt.Errorf("failed to read conn: %w", err)
		}
		buffer = append(buffer, temp[:n]...)

		// Check if we have received the complete HTTP request (ends with \r\n\r\n)
		if strings.HasSuffix(string(buffer), "\r\n\r\n") {
			break
		}
	}

	req := &Request{}
	if err := req.From(buffer); err != nil {
		return fmt.Errorf("failed to read request: %w", err)
	}

	fmt.Println(req)

	if err := handleRequest(ctx, req, conn); err != nil {
		return fmt.Errorf("failed to handle request: %w", err)
	}

	return nil
}

func handleRequest(ctx context.Context, req *Request, w io.Writer) error {
	switch req.Target {
	case "/":
		return handleRoot(ctx, req, w)
	default:
		return handleNotFound(ctx, req, w)
	}
}

func handleRoot(_ context.Context, req *Request, w io.Writer) error {
	resp := fmt.Sprintf("%s 200 OK%s", req.Version, CRLF)
	if _, err := w.Write([]byte(resp)); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}
	return nil
}

func handleNotFound(_ context.Context, req *Request, w io.Writer) error {
	resp := fmt.Sprintf("%s 404 Not Found%s", req.Version, CRLF)
	if _, err := w.Write([]byte(resp)); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}
	return nil
}
