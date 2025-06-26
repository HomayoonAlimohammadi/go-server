package main

import (
	"context"
	"fmt"
	"io"
	"strings"
)

func handleRoot(_ context.Context, req *Request, w io.Writer) error {
	if err := httpResponse(w, 200, ""); err != nil {
		return fmt.Errorf("failed to write HTTP response: %w", err)
	}
	return nil
}

func handleNotFound(_ context.Context, req *Request, w io.Writer) error {
	if err := httpResponse(w, 404, ""); err != nil {
		return fmt.Errorf("failed to write HTTP response: %w", err)
	}
	return nil
}

func handleEcho(_ context.Context, req *Request, w io.Writer) error {
	echo := strings.TrimPrefix(req.Target, "/echo/")
	if err := httpResponse(w, 200, echo); err != nil {
		return fmt.Errorf("failed to write HTTP response: %w", err)
	}
	return nil
}

func handleUserAgent(_ context.Context, req *Request, w io.Writer) error {
	if err := httpResponse(w, 200, req.Headers["User-Agent"]); err != nil {
		return fmt.Errorf("failed to write HTTP response: %w", err)
	}
	return nil
}
