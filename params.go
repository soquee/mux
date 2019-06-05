package mux

import (
	"net/http"
)

// Param returns the named route parameter from the requests context.
func Param(r *http.Request, name string) interface{} {
	return r.Context().Value(ctxParam(name))
}
