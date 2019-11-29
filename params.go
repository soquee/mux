package mux

import (
	"net/http"
)

// ctxParam is a type used for context keys that contain route parameters.
type ctxParam string

// ParamInfo represents a route parameter and related metadata.
type ParamInfo struct {
	// The parsed value of the parameter (for example int64(10))
	// If and only if no such parameter existed on the route, Value will be nil.
	Value interface{}
	// The raw value of the parameter (for example "10")
	Raw string
	// The name of the route component that the parameter was matched against (for
	// example "name" in "{name int}")
	Name string
	// Type type of the route component that the parameter was matched against
	// (for example "int" in "{name int}")
	Type string

	// offset is the number of the component in the route. Eg. a param foo in the
	// route /{foo int} has offset 1 (zero being the root node, which is never a
	// parameter).
	offset uint
}

// Param returns the named route parameter from the requests context.
func Param(r *http.Request, name string) ParamInfo {
	v := r.Context().Value(ctxParam(name))
	pinfo, _ := v.(ParamInfo)
	return pinfo
}
