// Package mux is a fast and safe HTTP request multiplexer.
//
// The multiplexer in this package is capable of routing based on request method
// and a fixed rooted path (/favicon.ico) or subtree (/images/) which may
// include typed path parameters and wildcards (see "URL Parameters").
//
//	m := mux.New(
//		mux.Handle(http.MethodGet, "/profile/{username string}", http.NotFoundHandler()),
//		mux.HandleFunc(http.MethodGet, "/profile", http.RedirectHandler("/profile/me", http.StatusPermanentRedirect)),
//		mux.Handle(http.MethodPost, "/logout", logoutHandler()),
//	)
//
// URL Parameters
//
// Routes registered on the multiplexer may contain variable path parameters
// that comprise an optional name, followed by a type.
//
//     /user/{id int}/edit
//
// Valid types include:
//
//     int    eg. -1, 1 (int64 in Go)
//     uint   eg. 0, 1 (uint64 in Go)
//     float  eg. 1, 1.123, -1.123 (float64 in Go)
//     string eg. anything ({string} is the same as {})
//     path   eg. files/123.png (must be the last path component)
//
// All numeric types are 64 bits wide.
// Parameters of type "path" match the remainder of the input path and therefore
// may only appear as the final component of a route:
//
//     /file/{p path}
//
// To retrieve the value of named path parameters see the Param function and the
// examples.
//
// Two paths with different typed variable parameters (including static routes)
// in the same position are not allowed.
// Attempting to register any two of the following routes will panic:
//
//     /user/{a int}/new
//     /user/{b int}/edit
//     /user/{float}/edit
//     /user/{b string}/edit
//     /user/me
package mux // import "code.soquee.net/mux"

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"strings"
)

// ctxRoute is a type used as the context key when storing a route on the HTTP
// context for future use.
type ctxRoute struct{}

const (
	typStatic = "static"
	typWild   = "path"
	typString = "string"
	typUint   = "uint"
	typInt    = "int"
	typFloat  = "float"
)

// ServeMux is an HTTP request multiplexer.
// It matches the URL of each incoming request against a list of registered
// patterns and calls the handler for the pattern that most closely matches the
// URL.
type ServeMux struct {
	node             node
	notFound         http.Handler
	methodNotAllowed http.Handler
	options          func(node) http.Handler
}

// New allocates and returns a new ServeMux.
func New(opts ...Option) *ServeMux {
	mux := &ServeMux{
		node: node{
			name:     "/",
			typ:      typStatic,
			handlers: make(map[string]http.Handler),
		},
		notFound: http.HandlerFunc(http.NotFound),
		methodNotAllowed: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}),
		options: defOptions,
	}
	for _, o := range opts {
		o(mux)
	}
	return mux
}

// ServeHTTP dispatches the request to the handler whose pattern most closely
// matches the request URL.
func (mux *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h, newReq := mux.handler(r)
	h.ServeHTTP(w, newReq)
}

// Handler returns the handler to use for the given request, consulting
// r.URL.Path.
// It always returns a non-nil handler and request.
//
// The path used is unchanged for CONNECT requests.
//
// If there is no registered handler that applies to the request, Handler
// returns a page not found handler.
// If a new request is returned it uses a context that contains any route
// parameters that were matched against the request path.
func (mux *ServeMux) Handler(r *http.Request) (http.Handler, *http.Request) {
	return mux.handler(r)
}

// handler returns the handler to use for the given request and a new request
// with parameters set on the context.
func (mux *ServeMux) handler(r *http.Request) (http.Handler, *http.Request) {
	// TODO: Add /tree to /tree/ redirect option and apply here.
	path := r.URL.Path

	// CONNECT requests are not canonicalized
	if r.Method != http.MethodConnect {
		path = cleanPath(r.URL.Path)
		if path != r.URL.Path {
			url := *r.URL
			url.Path = path
			return http.RedirectHandler(url.String(), http.StatusPermanentRedirect), r
		}
	}

	node := &mux.node
	path = strings.TrimPrefix(path, "/")

	// Requests for /
	if path == "" {
		h, ok := mux.node.handlers[r.Method]
		if !ok {
			switch {
			case r.Method == http.MethodOptions && mux.options != nil:
				return mux.options(mux.node), r
			case mux.methodNotAllowed != nil && (mux.options != nil || len(mux.node.handlers) > 0):
				return mux.methodNotAllowed, r
			}
			return mux.notFound, r
		}

		r = r.WithContext(context.WithValue(r.Context(), ctxRoute{}, mux.node.route))
		return h, r
	}

	offset := uint(1)

nodeloop:
	for node != nil {
		// If this is a variable route
		if len(node.child) == 1 && node.child[0].typ != typStatic {
			var part, remain string
			part, remain, r = node.child[0].match(path, offset, r)
			offset++

			// If the type doesn't match, we're done.
			if part == "" {
				return mux.notFound, r
			}

			// The variable route matched and it's the last thing in the path, so we
			// have our route:
			if remain == "" {
				h, ok := node.child[0].handlers[r.Method]
				if !ok {
					switch {
					case r.Method == http.MethodOptions && mux.options != nil:
						return mux.options(node.child[0]), r
					case mux.methodNotAllowed != nil && (mux.options != nil || len(node.child[0].handlers) > 0):
						return mux.methodNotAllowed, r
					}
					return mux.notFound, r
				}

				r = r.WithContext(context.WithValue(r.Context(), ctxRoute{}, node.child[0].route))
				return h, r
			}
			node = &node.child[0]
			path = remain
			continue
		}

		// If this is a static route
		for _, child := range node.child {
			var part, remain string
			part, remain, r = child.match(path, offset, r)
			offset++
			// The child did not match, so check the next.
			if part == "" {
				path = remain
				continue
			}

			// The child matched and was the last thing in the path, so we have our
			// route:
			if remain == "" {
				h, ok := child.handlers[r.Method]
				if !ok {
					switch {
					case r.Method == http.MethodOptions && mux.options != nil:
						return mux.options(child), r
					case mux.methodNotAllowed != nil && (mux.options != nil || len(mux.node.handlers) > 0):
						return mux.methodNotAllowed, r
					}
					return mux.notFound, r
				}

				r = r.WithContext(context.WithValue(r.Context(), ctxRoute{}, child.route))
				return h, r
			}

			// The child matched but was not the last one, move on to the next match.
			node = &child
			path = remain
			continue nodeloop
		}

		// No child matched.
		return mux.notFound, r
	}

	return mux.notFound, r
}

// parseParam returns a node with an empty handler from a path component.
func parseParam(pattern string) (name string, typ string) {
	// README:
	// The various checks in this function are a tad brittle and *order matters*
	// in subtle ways.
	// Be careful when refactoring this function.
	// that something is missing re-ordering these checks may result in panics.
	// Eventually we should build a proper tokenizer for this.

	// Static route components aren't patterns and must match exactly.
	if pattern[0] != '{' || pattern[len(pattern)-1] != '}' {
		return pattern, typStatic
	}

	// {} is an unnamed variable (it matches any single path component)
	if len(pattern) == 2 {
		return "", typString
	}

	// Variable matches ("{name type}" or "{type}")
	idx := strings.IndexByte(pattern, ' ')
	if idx == -1 {
		idx = 0
	}
	typ = pattern[idx+1 : len(pattern)-1]
	if idx == 0 {
		idx = 1
	}

	switch typ {
	case typInt, typUint, typFloat, typString, typWild:
		return pattern[1:idx], typ
	}
	panic(fmt.Sprintf("invalid type: %q", typ))
}

func nextPart(path string) (string, string) {
	idx := strings.IndexByte(path, '/')
	if idx == -1 {
		return path, ""
	}
	return path[:idx], path[idx+1:]
}

// Code below this line was taken from the Go source and is used under the terms
// of Go's BSD license (see the file LICENSE-GO). Its copyright statement is
// below:
//
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Return the canonical path for p, eliminating . and .. elements.
func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	np := path.Clean(p)
	// path.Clean removes trailing slash except for root;
	// put the trailing slash back if necessary.
	if p[len(p)-1] == '/' && np != "/" {
		np += "/"
	}
	return np
}
