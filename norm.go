package mux

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

var (
	errNoRoute = errors.New("mux: no route was found in the context")
	errNoParam = errors.New("mux: context was missing an expected parameter")
)

// WithParam returns a shallow copy of r with a new context that shadows the
// given route parameter.
// If the parameter does not exist, the original request is returned unaltered.
//
// Because WithParam is used to normalize request parameters after the route
// has already been resolved, all replaced parameters are of type string.
func WithParam(r *http.Request, name, val string) *http.Request {
	pinfo := Param(r, name)
	if pinfo.Value == nil {
		return r
	}

	pinfo.Value = val
	pinfo.Raw = val
	pinfo.Type = typString
	return r.WithContext(context.WithValue(r.Context(), ctxParam(name), pinfo))
}

// Path returns the request path by applying the route parameters found in the
// context to the route used to match the given request.
// This value may be different from r.URL.Path if some form of normalization has
// been applied to a route parameter, in which case the user may choose to issue
// a redirect to the canonical path.
func Path(r *http.Request) (string, error) {
	route := r.Context().Value(ctxRoute{}).(string)
	if route == "" {
		return "", errNoRoute
	}
	hasTrailingSlash := strings.HasSuffix(route, "/")
	oldPath := strings.TrimPrefix(r.URL.Path, "/")

	var canonicalPath strings.Builder
	// Give us a comfortable capacity so that we have to resize the buffer less
	// often.
	canonicalPath.Grow(len(route))

	for {
		var component, pathComponent string
		pathComponent, oldPath = nextPart(oldPath)

		component, route = nextPart(route)
		if component == "" {
			// Add back any trailing slash consumed by nextPart.
			if hasTrailingSlash {
				err := canonicalPath.WriteByte('/')
				if err != nil {
					return "", err
				}
			}
			break
		}
		err := canonicalPath.WriteByte('/')
		if err != nil {
			return "", err
		}
		name, typ := parseParam(component)
		switch {
		case typ == typStatic:
			_, err = canonicalPath.WriteString(name)
			if err != nil {
				return "", err
			}
		case name == "":
			_, err = canonicalPath.WriteString(pathComponent)
			if err != nil {
				return "", err
			}
		default:
			pinfo := Param(r, name)
			if pinfo.Value == nil {
				return "", errNoParam
			}
			_, err = canonicalPath.WriteString(pinfo.Raw)
			if err != nil {
				return "", err
			}
		}
	}

	return canonicalPath.String(), nil
}
