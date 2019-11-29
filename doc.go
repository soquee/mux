// Package mux is a fast and safe HTTP request multiplexer.
//
// The multiplexer in this package is capable of routing based on request method
// and a fixed rooted path (/favicon.ico) or subtree (/images/) which may
// include typed path parameters and wildcards (see "URL Parameters").
//
//	m := mux.New(
//		mux.Handle(http.MethodGet, "/profile/{username string}", http.NotFoundHandler()),
//		mux.HandleFunc(http.MethodGet, "/profile", http.RedirectHandler("/profile/me", http.StatusPermanentRedirect)),
//		mux.Handle(http.MethodPost, "/logout", logoutHandler()),
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
