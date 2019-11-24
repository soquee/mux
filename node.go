package mux

import (
	"context"
	"net/http"
	"strconv"
)

type node struct {
	name     string
	typ      string
	handlers map[string]http.Handler

	child []node
}

func (n *node) match(path string, offset uint, r *http.Request) (part string, remain string, req *http.Request) {
	// Nil nodes never match.
	if n == nil {
		return "", "", r
	}

	part, remain = nextPart(path)
	switch n.typ {
	case typStatic:
		if n.name == part {
			return part, remain, r
		}
		return "", path, r
	case typString:
		r = addValue(r, n.name, n.typ, part, offset, part)
		return part, remain, r
	case typWild:
		r = addValue(r, n.name, n.typ, path, offset, path)

		// If we're the last node in the route, consume the remainder of the path.
		if len(n.child) == 0 {
			return path, "", r
		}
		return part, remain, r
	case typUint:
		v, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return "", path, r
		}
		r = addValue(r, n.name, n.typ, part, offset, v)
		return part, remain, r
	case typInt:
		v, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return "", path, r
		}
		r = addValue(r, n.name, n.typ, part, offset, v)
		return part, remain, r
	case typFloat:
		v, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return "", path, r
		}
		r = addValue(r, n.name, n.typ, part, offset, v)
		return part, remain, r
	}
	panic("unknown type")
}

func addValue(r *http.Request, name, typ, raw string, offset uint, val interface{}) *http.Request {
	if name != "" {
		pinfo := ParamInfo{
			Value:  val,
			Raw:    raw,
			Name:   name,
			Type:   typ,
			Offset: offset,
		}
		return r.WithContext(context.WithValue(r.Context(), ctxParam(name), pinfo))
	}
	return r
}
