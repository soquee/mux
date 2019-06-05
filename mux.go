// Package mux is a fast and safe HTTP multiplexer.
//
// URL Parameters
//
// Routes registered on the multiplexer may contain variable path parameters
// that comprise an optional name, followed by a type.
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
// A non-path typed variable parameter may appear anywhere in the path and match
// a single path component:
//
//     /user/{id int}/edit
//
// Parameters of type "path" match the remainder of the input path and therefore
// may only appear as the final component of a route:
//
//     /file/{p path}
//
// Two paths with different typed variable parameters (including static routes)
// in the same position are not allowed.
// Attempting to register any two of the following routes will panic:
//
//     /user/{a int}/new
//     /user/{b int}/edit
//     /user/{b string}/edit
//     /user/me
package mux // import "code.soquee.net/mux"

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path"
	"strconv"
	"strings"
)

const (
	emptyPanic  = "invalid empty pattern"
	panicNoRoot = "all handlers must start with /"
	typStatic   = "static"
	typWild     = "path"
	typString   = "string"
	typUint     = "uint"
	typInt      = "int"
	typFloat    = "float"
)

// ServeMux is an HTTP request multiplexer.
// It matches the URL of each incoming request against a list of registered
// patterns and calls the handler for the pattern that most closely matches the
// URL.
type ServeMux struct {
	node
	notFound http.Handler
}

// ServeHTTP dispatches the request to the handler whose pattern most closely
// matches the request URL.
func (mux *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h, newReq := mux.handler(r)
	h.ServeHTTP(w, newReq)
}

// stripHostPort returns h without any trailing ":<port>".
func stripHostPort(h string) string {
	// If no port on host, return unchanged
	if strings.IndexByte(h, ':') == -1 {
		return h
	}
	host, _, err := net.SplitHostPort(h)
	if err != nil {
		return h // on error, return unchanged
	}
	return host
}

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

// Handler returns the handler to use for the given request, consulting
// r.URL.Path.
// It always returns a non-nil handler.
//
// The path and host are used unchanged for CONNECT requests.
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
	// TODO: use host
	host := r.Host
	_ = host
	path := r.URL.Path

	// CONNECT requests are not canonicalized
	if r.Method != "CONNECT" {
		// All other requests have any port stripped and path cleaned
		// before passing to mux.handler.
		host = stripHostPort(r.Host)
		path = cleanPath(r.URL.Path)
		if path != r.URL.Path {
			url := *r.URL
			url.Path = path
			return http.RedirectHandler(url.String(), http.StatusPermanentRedirect), r
		}
	}

	// TODO: add host based matching and check it here.
	node := &mux.node
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		if node.handler == nil {
			return mux.notFound, r
		}
		return node.handler, r
	}

nodeloop:
	for node != nil {
		// If this is a variable route,
		if len(node.child) == 1 && node.child[0].typ != typStatic {
			var part, remain string
			part, remain, r = node.child[0].match(path, r)

			// If the type doesn't match, we're done.
			if part == "" {
				return mux.notFound, r
			}

			// The variable route matched and it's the last thing in the path, so we
			// have our route:
			if remain == "" {
				if node.child[0].handler == nil {
					return mux.notFound, r
				}
				return node.child[0].handler, r
			}
			node = &node.child[0]
			path = remain
			continue
		}

		// If this is a static route
		for _, child := range node.child {
			var part, remain string
			part, remain, r = child.match(path, r)
			// The child did not match, so check the next.
			if part == "" {
				path = remain
				continue
			}

			// The child matched and was the last thing in the path, so we have our
			// route:
			if remain == "" {
				if child.handler == nil {
					return mux.notFound, r
				}
				return child.handler, r
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

type node struct {
	name    string
	typ     string
	handler http.Handler

	child []node
}

// ctxParam is a type used for context keys that contain route parameters.
type ctxParam string

func addValue(r *http.Request, name string, val interface{}) *http.Request {
	if name != "" {
		return r.WithContext(context.WithValue(r.Context(), ctxParam(name), val))
	}
	return r
}

// Param returns the named route parameter from the requests context.
func Param(r *http.Request, name string) interface{} {
	return r.Context().Value(ctxParam(name))
}

// TODO: take a context and put parameters into it.
func (n *node) match(path string, r *http.Request) (part string, remain string, req *http.Request) {
	// Nil nodes never match.
	if n == nil {
		return "", "", r
	}

	// wildcards are a special case that always match the entire remainder of the
	// path.
	if n.typ == typWild {
		r = addValue(r, n.name, path)
		return path, "", r
	}

	part, remain = nextPart(path)
	switch n.typ {
	case typStatic:
		if n.name == part {
			return part, remain, r
		}
		return "", path, r
	case typString:
		r = addValue(r, n.name, part)
		return part, remain, r
	case typUint:
		v, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return "", path, r
		}
		r = addValue(r, n.name, v)
		return part, remain, r
	case typInt:
		v, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return "", path, r
		}
		r = addValue(r, n.name, v)
		return part, remain, r
	case typFloat:
		v, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return "", path, r
		}
		r = addValue(r, n.name, v)
		return part, remain, r
	}
	panic("unknown type")
}

// New allocates and returns a new ServeMux.
func New(opts ...Option) *ServeMux {
	mux := &ServeMux{
		node: node{
			name:    "/",
			typ:     typStatic,
			handler: nil,
		},
		notFound: http.HandlerFunc(http.NotFound),
	}
	for _, o := range opts {
		o(mux)
	}
	return mux
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

// Handle registers the handler for the given pattern.
// If a handler already exists for pattern, Handle panics.
func HandleFunc(r string, h http.HandlerFunc) Option {
	return Handle(r, h)
}

// Handle registers the handler for the given pattern.
// If a handler already exists for pattern, Handle panics.
func Handle(r string, h http.Handler) Option {
	if !strings.HasPrefix(r, "/") {
		panic(panicNoRoot)
	}
	r = r[1:]

	const (
		alreadyRegistered = "route already registered for /%s"
	)

	return func(mux *ServeMux) {
		pointer := &mux.node

		// If we're registering a root handler
		if r == "" {
			// If it exists already
			if pointer.handler != nil {
				panic(fmt.Sprintf(alreadyRegistered, r))
			}
			pointer.handler = h
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
						if child.handler == nil {
							pointer.child[i].handler = h
							continue pathloop
						} else {
							// If one already exists and this is the path we were trying to
							// register, panic.
							panic(fmt.Sprintf(alreadyRegistered, r))
						}
					}

					pointer = &pointer.child[i]
					continue pathloop
				}
			}

			// Not found at his level. Append new node.
			n := node{
				name: name,
				typ:  typ,
			}
			if remain == "" {
				n.handler = h
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
		panic(emptyPanic)
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
