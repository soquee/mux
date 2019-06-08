package mux

import (
	"net/http"
)

// ctxParam is a type used for context keys that contain route parameters.
type ctxParam string

// ParamInfo represents a route parameter and related metadata.
type ParamInfo struct {
	// The parsed value of the parameter (for example int64(10))
	Value interface{}
	// The raw value of the parameter (for example "10")
	Raw string
	// The name of the route component that the parameter was matched against (for
	// example "name" in "{name int}")
	Name string
	// Type type of the route component that the parameter was matched against
	// (for example "int" in "{name int}")
	Type string
	// The offset in the path where this parameter was found (for example if "10"
	// is parsed out of the path "/10" the offset is 1)
	Offset uint
}

// Param returns the named route parameter from the requests context.
func Param(r *http.Request, name string) (pinfo ParamInfo, ok bool) {
	v := r.Context().Value(ctxParam(name))
	if v == nil {
		return ParamInfo{}, false
	}
	return v.(ParamInfo), true
}
