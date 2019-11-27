package mux

import (
	"fmt"
	"net/http"
	"strings"
)

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

// Options changes the ServeMux's default OPTIONS request handling behavior.
// If you do not want options handling by default, set f to "nil".
//
// Registering handlers for OPTIONS requests on a specific path always overrides
// the default handler.
func Options(f func([]string) http.Handler) Option {
	return func(mux *ServeMux) {
		if f == nil {
			mux.options = nil
			return
		}

		mux.options = func(n node) http.Handler {
			var verbs []string
			for v := range n.handlers {
				verbs = append(verbs, v)
			}
			return f(verbs)
		}
	}
}

// MethodNotAllowed sets the default handler to call when a path is matched to a
// route, but there is no handler registered for the specific method.
//
// By default, http.Error with http.StatusMethodNotAllowed is used.
func MethodNotAllowed(h http.Handler) Option {
	return func(mux *ServeMux) {
		mux.methodNotAllowed = h
	}
}

// HandleFunc registers the handler for the given pattern.
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
			pointer.route = r
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
							pointer.child[i].route = r
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
				n.route = r
				n.handlers[method] = h
			}

			pointer.child = append(pointer.child, n)
			pointer = &pointer.child[len(pointer.child)-1]
		}
	}
}
