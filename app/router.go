package main

import (
	"context"
	"io"
	"log"
	"strings"
)

type match struct {
	method  string
	prefix  string
	handler handleFunc
}

type router struct {
	routes   []match
	notFound handleFunc
}

func NewRouter() *router {
	return &router{notFound: handleNotFound}
}

type handleFunc func(context.Context, *Request, io.Writer) error

func (r *router) register(method string, prefix string, handler handleFunc) {
	r.routes = append(r.routes, match{method: method, prefix: prefix, handler: handler})
}

func (r *router) Route(ctx context.Context, req *Request, w io.Writer) error {
	for _, m := range r.routes {
		if req.Method == m.method && strings.HasPrefix(req.Target, m.prefix) {
			log.Printf("Matched method=%s prefix=%s", m.method, m.prefix)
			return m.handler(ctx, req, w)
		}
	}
	log.Println("Did not match any route")
	return r.notFound(ctx, req, w)
}
