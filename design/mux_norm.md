# Proposal: Normalization of Route Parameters

**Author(s):** Sam Whited  
**Last updated:** 2020-01-02  
**Status:** implemented

## Abstract

A proposal to simplify normalization of route parameters by including such
functionality in the libraries public API instead of requiring that users
implement it themselves.

## Background

On the popular code hosting service GitHub the following URLs
resolve to the same resource:

 - https://github.com/mellium/xmpp
 - https://github.com/Mellium/xmpp
 - https://github.com/melLiUm/xMPP

It is generally accepted that this is not a good idea [citation needed].
Instead, it is desirable to have a single canonical URL per resource and, if
necessary, redirect non-canonical URLs to their canonical form.
Writing code to do this is not difficult, but it can be tedious and error prone.
Having an HTTP multiplexer specifically provide an API for route normalization
could make this easier, and hopefully more commonplace.


## Requirements

The [Go] multiplexer [`code.soquee.net/mux`] supports matching typed route
parameters that are stored on Go's request [context], and strives to export a
minimal public API.
Changes adding support for path normalization must meet the following design
requirements:

 - Introduce as few new public identifiers as possible
 - Allow replacing multiple route parameters with normalized values given a
   single [`Request`] value
 - Do not perform any mutation of data that could alter other code's view of
   the request, route, or request parameters
 - Do not require that the module (which practices [semver]) undergo a major
   version bump


## Proposal

The following API, which introduces two new identifiers that must meet the mux
module's compatibility guarantees, is proposed:

```go
// WithParam returns a shallow copy of r with a new context that shadows the
// given route parameter. If the parameter does not exist, the original request
// is returned unaltered.
func WithParam(r *http.Request, name, val string) *http.Request {/* … */}

// Path returns the request path by applying the route parameters found in the
// context to the route used to match the given request. This value may be
// different from r.URL.Path if some form of normalization has been applied to a
// route parameter, in which case the user may choose to issue a redirect to the
// canonical path.
func Path(r *http.Request) (string, error) {/* … */}
```

Because normalization of route parameters happens after route matching, the type
is no longer useful and `WithParam` may always set a string type.
If this assumption ends up being wrong, the `val` parameter could be changed to
an empty interface and `WithParam` could do runtime type checking, however,
this would require some sort of error return value in the case of a type that
is not supported and complicates use of the API.
The trade-offs will hopefully become more apparent before version 1.0 when this
API is locked in.

Implementing this API makes the [`ParamInfo.Offset`] field unnecessary.
Because the module has not yet reached version 1.0, this field can be removed
without requiring a major version bump, or it can be left in to prevent
breakage at our discretion.

From the users perspective normalizing a GitHub style URL with the following
route:

    github.com/{username string}/{repo string}

would look like the following:

```go
username := mux.Param(req, "username")
req = mux.WithParam(req, strings.ToLower(username))

repo := mux.Param(req, "repo")
req = mux.WithParam(ctx, strings.ToLower(repo.Raw))

normPath := mux.Path(req)
```

[Go]: https://golang.org/
[context]: https://golang.org/pkg/context/
[`soquee.net/mux`]: https://code.soquee.net/mux/
[`Request`]: https://golang.org/pkg/net/http/#Request
[`ParamInfo.Offset`]: https://godoc.org/code.soquee.net/mux#ParamInfo.Offset
[semver]: https://semver.org/
