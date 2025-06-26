package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		log.Println("Failed to bind to port 4221: ", err.Error())
		os.Exit(1)
	}

	tcpL, ok := l.(*net.TCPListener)
	if !ok {
		log.Println("Failed to convert to TCP listener")
		os.Exit(1)
	}

	var dir string
	flag.StringVar(&dir, "directory", "/tmp/", "Directory to look for the files")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	srv := NewServer(dir, tcpL, shutdownCh)
	srv.Register(http.MethodGet, "/files", srv.handleFiles)
	srv.Register(http.MethodGet, "/user-agent", srv.handleUserAgent)
	srv.Register(http.MethodGet, "/echo", srv.handleEcho)
	srv.Register(http.MethodGet, "/", srv.handleRoot)

	if err := srv.Start(ctx); err != nil {
		log.Println("Failed to start server: ", err.Error())
	}
}
