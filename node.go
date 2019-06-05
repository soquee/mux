package mux

import (
	"context"
	"net/http"
	"strconv"
)

type node struct {
	name    string
	typ     string
	handler http.Handler

	child []node
}

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

func addValue(r *http.Request, name string, val interface{}) *http.Request {
	if name != "" {
		return r.WithContext(context.WithValue(r.Context(), ctxParam(name), val))
	}
	return r
}
