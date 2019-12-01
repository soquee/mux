// Package mux is a fast and safe HTTP request multiplexer.
//
// The multiplexer in this package is capable of routing based on request method
// and a fixed rooted path (/favicon.ico) or subtree (/images/) which may
// include typed path parameters and wildcards.
//
//	serveMux := mux.New(
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
// Two paths with different typed variable parameters (including static routes)
// in the same position are not allowed.
// Attempting to register any two of the following routes will panic:
//
//     /user/{a int}/new
//     /user/{b int}/edit
//     /user/{float}/edit
//     /user/{b string}/edit
//     /user/me
//
// This is to prevent a common class of bug where a static route conflicts with
// a path parameter and it is not clear which should be selected.
// For example, if the route /users/new and /users/{username string} could be
// registered at the same time and someone attempts to register the user "new",
// it might make the new user page impossible to visit, or break the new users
// profile.
// Disallowing conflicting routes keeps things simple and eliminates this class
// of issues.
//
// When a route is matched, the value of each named path parameter is stored on
// the request context.
// To retrieve the value of named path parameters from within a handler, the
// Param function can be used.
//
//    pinfo := mux.Param(req, "username")
//    fmt.Println("Got username:", pinfo.Raw)
//
// For more information, see the ParamInfo type and the examples.
//
// Normalization
//
// It's common to normalize routes on HTTP servers.
// For example, a username may need to match the Username Case Mapped profile of
// PRECIS (RFC 8265), or the name of an identifier may need to always be lower
// cased.
// This can be tedious and error prone even if you have enough information to
// figure out what path components need to be replaced, and many HTTP routers
// don't even give you enough information to match path components to their
// original route parameters.
// To make this easier, this package provides the Path and WithParam functions.
// WithParam is used to attach a new context to the context tree with
// replacement values for existing route parameters, and Path is used to
// re-render the path from the original route using the request context.
// If the resulting path is different from req.URL.Path, a redirect can be
// issued or some other corrective action can be applied.
//
//	serveMux := mux.New(
//		mux.HandleFunc(http.MethodGet, "/profile/{username string}", func(w http.ResponseWriter, r *http.Request) {
//			username := mux.Param(r, "username")
//			// golang.org/x/text/secure/precis
//			normalized, err := precis.UsernameCaseMapped.String(username.Raw)
//			if err != nil {
//					…
//			}
//
//			if normalized != username.Raw {
//				r = mux.WithParam(r, username.Name, normalized)
//				newPath, err := mux.Path(r)
//				if err != nil {
//					…
//				}
//				http.Redirect(w, r, newPath, http.StatusPermanentRedirect)
//				return
//			}
//
//			fmt.Fprintln(w, "The canonical username is:", username.Raw)
//		}),
//	)
//
// For more information, see the examples.
package mux // import "code.soquee.net/mux"
