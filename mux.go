// Package mux is a fast and safe HTTP request multiplexer.
//
// The muxer in this package is capable of routing based on request method and a
// fixed rooted path (/favicon.ico) or subtree (/images/) which may include
// typed path parameters and wildcards (see "URL Parameters").
//
//	m := mux.New(
//		mux.HandleFunc(http.MethodGet, "/profile/{username string}", http.NotFoundHandler())
//		mux.HandleFunc(http.MethodGet, "/profile", http.RedirectHandler("/profile/me", http.StatusPermanentRedirect))
//		mux.HandleFunc(http.MethodPost, "/logout", logoutHandler())
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
	"fmt"
	"net/http"
	"path"
	"strings"
)

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
	node
	notFound http.Handler
	options  func(node) http.Handler
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
		options:  defOptions,
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
// It always returns a non-nil handler.
//
// The path used is unchanged for CONNECT requests.
//
// If there is no registered handler that applies to the request, Handler
// returns a ``page not found'' handler and an empty pattern.
// The new request uses a context that contains any route parameters that were
// matched against the request path.
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
			// TODO: method not supported vs not found config
			if r.Method == http.MethodOptions && mux.options != nil {
				return mux.options(mux.node), r
			}
			return mux.notFound, r
		}
		return h, r
	}

	offset := uint(1)

nodeloop:
	for node != nil {
		// If this is a variable route
		if len(node.child) == 1 && node.child[0].typ != typStatic {
			var part, remain string
			part, remain, r = node.child[0].match(path, offset, r)
			offset += uint(len(part)) + 1

			// If the type doesn't match, we're done.
			if part == "" {
				return mux.notFound, r
			}

			// The variable route matched and it's the last thing in the path, so we
			// have our route:
			if remain == "" {
				h, ok := node.child[0].handlers[r.Method]
				if !ok {
					// TODO: method not supported vs not found config
					if r.Method == http.MethodOptions && mux.options != nil {
						return mux.options(node.child[0]), r
					}
					return mux.notFound, r
				}
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
			offset += uint(len(part)) + 1
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
					// TODO: method not supported vs not found config
					if r.Method == http.MethodOptions && mux.options != nil {
						return mux.options(child), r
					}
					return mux.notFound, r
				}
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

// Option is used to configure a ServeMux.
type Option func(*ServeMux)

// NotFound sets the handler to use when a request does not have a registered
// route.
//
// If the provided handler does not set the status code, it is set to 404 (Page
// Not Found) by default instead of 200.
// If the provided handler explicitly sets the status by calling
// "http.ResponseWriter".WriteHeader, that status code is used instead.
func NotFound(h http.Handler) Option {
	return func(mux *ServeMux) {
		mux.notFound = notFoundHandler(h)
	}
}

// The ServeMux handles OPTIONS requests by default. If you do not want this
// behavior, set f to "nil".
//
// Registering handlers for OPTIONS requests on a specific path always overrides
// the default handler.
func DefaultOptions(f func([]string) http.Handler) Option {
	return func(mux *ServeMux) {
		if f == nil {
			mux.options = nil
			return
		}

		mux.options = func(n node) http.Handler {
			var verbs []string
			for v, _ := range n.handlers {
				verbs = append(verbs, v)
			}
			return f(verbs)
		}
	}
}

// Handle registers the handler for the given pattern.
// If a handler already exists for pattern, Handle panics.
func HandleFunc(method, r string, h http.HandlerFunc) Option {
	return Handle(method, r, h)
}

// Handle registers the handler for the given pattern.
// If a handler already exists for pattern, Handle panics.
func Handle(method, r string, h http.Handler) Option {
	method = strings.ToUpper(method)
	if rr := cleanPath(r); rr != r {
		panic(fmt.Sprintf("route %q is unclean, make sure it is rooted and remove any ., .., or //", r))
	}
	r = r[1:]

	const (
		alreadyRegistered = "route already registered for %s /%s"
	)

	return func(mux *ServeMux) {
		pointer := &mux.node

		// If we're registering a root handler
		if r == "" {
			// If it exists already
			if _, ok := pointer.handlers[method]; ok {
				panic(fmt.Sprintf(alreadyRegistered, method, r))
			}
			pointer.handlers[method] = h
			return
		}

	pathloop:
		for part, remain := nextPart(r); remain != "" || part != ""; part, remain = nextPart(remain) {
			name, typ := parseParam(part)

			if typ == typWild && remain != "" {
				panic(fmt.Sprintf("wildcards must be the last component in a route: /%s", r))
			}

			// If there are already children, check that this one is compatible with
			// them.
			if len(pointer.child) > 0 {
				child := pointer.child[0]
				switch {
				// All non static routes must have the same type and name.
				case typ != typStatic && child.typ != typ:
					panic(fmt.Sprintf("conflicting type found, {%s %s} in route %q conflicts with existing registration of {%s %s}", name, typ, r, pointer.child[0].name, pointer.child[0].typ))
				case typ != typStatic && child.name != name:
					panic(fmt.Sprintf("conflicting variable name found, {%s %s} in route %q conflicts with existing registration of {%s %s}", name, typ, r, pointer.child[0].name, pointer.child[0].typ))
				// All static routes must have the same type.
				case typ == typStatic && child.typ != typ:
					panic(fmt.Sprintf("conflicting type found, {%s %s} in route %q conflicts with existing registration of {%s %s}", name, typ, r, pointer.child[0].name, pointer.child[0].typ))
				}
			}

			// Check if a node already exists in the tree with this name.
			for i, child := range pointer.child {
				if child.name == name {
					if remain == "" {
						// If this is the path we want to register and no handler has been
						// registered for it, add one:
						if _, ok := child.handlers[method]; !ok {
							pointer.child[i].handlers[method] = h
							continue pathloop
						} else {
							// If one already exists and this is the path we were trying to
							// register, panic.
							panic(fmt.Sprintf(alreadyRegistered, method, r))
						}
					}

					pointer = &pointer.child[i]
					continue pathloop
				}
			}

			// Not found at his level. Append new node.
			n := node{
				name:     name,
				typ:      typ,
				handlers: make(map[string]http.Handler),
			}
			if remain == "" {
				n.handlers[method] = h
			}

			pointer.child = append(pointer.child, n)
			pointer = &pointer.child[len(pointer.child)-1]
		}
	}
}

type route struct{}

// parseParam returns a node with an empty handler from a path component.
func parseParam(pattern string) (name string, typ string) {
	// README:
	// The various checks in this function are a tad brittle and *order matters*
	// in subtle ways.
	// Be careful when refactoring this function.
	// that something is missing re-ordering these checks may result in panics.
	// Eventually we should build a proper tokenizer for this.

	// We should never be passed an empty pattern.
	// If we get one, it's a bug.
	if pattern == "" {
		panic("invalid empty pattern")
	}

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
