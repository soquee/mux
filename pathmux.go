// Package mux is a fast and safe HTTP multiplexer.
//
// URL Parameters
//
// Routes registered on the multiplexer may contain variable path parameters that
// comprise an optional name, followed by a type.
//
// Valid types include:
//
//     int    eg. -1, 1
//     uint   eg. 0, 1
//     float  eg. 1, 1.123, -1.123
//     string eg. anything ({string} is the same as {})
//     path   eg. files/123.png
//
// All numeric types are 64 bits wide.
// A non-path typed variable parameter may appear anywhere in the path and match
// a single path component:
//
//     /user/{id int}/edit
//
// Parameters of type "path" match the remainder of the input path and therefore
// may only appear as the final component of a route:
//     /file/{p path}
//
// When constructing routes a variable path parameter and a static path
// parameter may appear in the same location.
// In this case, the static parmeter takes priority and if the match fails later
// on in the path, no backtracking occures.
// For example, if the following routes are registered:
//
//     /user/me/edit
//     /user/{string}/edit
//     /user/{string}/new
//
// The path /user/me/edit will always match the first handler and the path
// /user/me/new will not match any handler (because the first two elements will
// match the "user" and "me" components of the first route, and then fail on the
// "edit" / "new" comparison).
//
// Two paths with different typed variable parameters in the same position are
// not allowed.
// Attempting to register the following routes will panic:
//
//     /user/{int}/edit
//     /user/{string}/edit
package mux

import (
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
	// TODO: Why is this in the standard library version?
	// if r.RequestURI == "*" {
	// 	if r.ProtoAtLeast(1, 1) {
	// 		w.Header().Set("Connection", "close")
	// 	}
	// 	w.WriteHeader(StatusBadRequest)
	// 	return
	// }
	mux.Handler(r).ServeHTTP(w, r)
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
func (mux *ServeMux) Handler(r *http.Request) http.Handler {
	// CONNECT requests are not canonicalized.
	if r.Method == "CONNECT" {
		// TODO: Add /tree to /tree/ redirect option and apply here.
		return mux.handler(r.Host, r.URL.Path)
	}

	// All other requests have any port stripped and path cleaned
	// before passing to mux.handler.
	host := stripHostPort(r.Host)
	path := cleanPath(r.URL.Path)

	// TODO: Add /tree to /tree/ redirect option and apply here.

	if path != r.URL.Path {
		url := *r.URL
		url.Path = path
		return http.RedirectHandler(url.String(), http.StatusPermanentRedirect)
	}

	return mux.handler(host, r.URL.Path)
}

func (mux *ServeMux) handler(host, path string) http.Handler {
	// TODO: add host based matching and check it here.
	node := &mux.node
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		if node.handler == nil {
			return mux.notFound
		}
		return node.handler
	}

	// Match static route parameters first.
nodeloop:
	for node != nil {
		for _, child := range node.child {
			part, remain := child.match(path)
			// The child did not match, so check the next.
			if part == "" {
				path = remain
				continue
			}

			// The child matched and was the last thing in the path, so we have our
			// route:
			if remain == "" {
				if child.handler == nil {
					return mux.notFound
				}
				return child.handler
			}

			// The child matched but was not the last one, move on to the next match.
			node = &child
			path = remain
			continue nodeloop
		}

		// No static routes matched, check if there is a variable route.
		part, remain := node.varchild.match(path)

		// If there is no variable route or there was one but the type doesn't
		// match, we're done.
		if part == "" {
			return mux.notFound
		}

		// The variable route matched and it's the last thing in the path, so we
		// have our route:
		if remain == "" {
			if node.varchild.handler == nil {
				return mux.notFound
			}
			return node.varchild.handler
		}

		node = node.varchild
	}

	return mux.notFound
}

type node struct {
	name    string
	typ     string
	handler http.Handler

	varchild *node
	child    []node
}

// TODO: take a context and put parameters into it.
func (n *node) match(path string) (match string, remain string) {
	// Nil nodes never match.
	if n == nil {
		return "", ""
	}

	// wildcards are a special case that always match the entire remainder of the
	// path.
	if n.typ == typWild {
		return path, ""
	}

	part, remain := nextPart(path)
	switch n.typ {
	case typStatic:
		if n.name == part {
			return part, remain
		}
		return "", path
	case typString:
		return part[1:], remain
	case typUint:
		_, err := strconv.ParseUint(path, 10, 64)
		if err != nil {
			return "", path
		}
		return part, remain
	case typInt:
		_, err := strconv.ParseInt(path, 10, 64)
		if err != nil {
			return "", path
		}
		return part, remain
	case typFloat:
		_, err := strconv.ParseFloat(path, 64)
		if err != nil {
			return "", path
		}
		return part, remain
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
// The handler should set the status code to 404 (Page Not Found).
func NotFound(h http.Handler) Option {
	return func(mux *ServeMux) {
		mux.notFound = h
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

			// If this is a variable type there can only be one per level.
			if typ != typStatic {
				// If a variable is already registered, panic.
				if pointer.varchild != nil {
					panic(fmt.Sprintf("conflicting variable type found, {%s %s} in route %q conflicts with existing registration of {%s %s}", name, typ, r, pointer.varchild.name, pointer.varchild.typ))
				}

				pointer.varchild = &node{
					typ:  typ,
					name: name,
				}
				if remain == "" {
					pointer.varchild.handler = h
				}
				pointer = pointer.varchild
				continue
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

	// Wildcards ("{*}" or "{*name}") match the rest of the path
	if pattern[1] == '*' {
		return strings.TrimSpace(pattern[2 : len(pattern)-1]), typWild
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
	case typInt, typUint, typFloat, typString:
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
