# Proposal: Relaxed Path Matching

**Author(s):** Sam Whited  
**Last updated:** 2020-01-02  
**Status:** thinking

## Abstract

A design is proposed for relaxing the rule that requires `path` typed route
parameters to only exist as the last component in a route.
This would allow for greater flexibility in route construction and fewer special
cases in the type system.

## Background

In a typed HTTP router the "path" type always matches the remainder of the
input path. For example, the route:

    /files/{p path}

Matches:

- `/files/img.png`
- `/files/myalbum/img.png`

Right now it must always be the last item in any route otherwise creating the
router fails.

I propose relaxing this restriction by removing the special case that only
allows path type parameters at the end of routes, and making them match exactly
one path component if they are not at the end of the route, or the remainder of
the path if they are.


## Requirements

 - Do not break existing end-of-path matching support
 - No new public types or identifiers
 - Ability to remove the special case in the type system to panic if a `path`
   type route parameter is registered in an incorrect way


## Proposal

This proposal creates a new distinction between the *value* of a route parameter
and the matched path component.
Matching a `path` typed route parameter, the value would now be the remainder of
the path, but it would match against any single path component similar to a
string match:

    /albums/{p path}/cover.png

Would match:

- /albums/myalbum/cover.png
- /albums/whatever/cover.png

But would not match:

- /albums/foo/bar/cover.png

While the previous example with a `path` typed parameter at the end of the route
would behave exactly the same way as it previously did.

The matching behaves the same as using the string type, except that the value
of p would be "myalbum/cover.png" and "whatever/cover.png" instead of just
"myalbum" or "whatever" as it would be if the route had been `/albums/{p
string}/cover.png`.

An empty route parameter (`{}`) is the same as an unnamed parameter of type
string, ie. `{string}` and matches any single path component (but its value is
not saved). This means that with this restriction if we wanted to match any
path with two components, for example, Git repos in a GitHub style URL, instead
of doing:

    /{username string}/{repo string}

And then getting the values of username and repo and joining them together, we
could do something like the following to match any path with 2 components and
pull out the path directly without any extra logic in our HTTP handler:

    /{repo path}/{}

This would match:

- `/samwhited/mux`
- `/foo/bar`
- etc.
