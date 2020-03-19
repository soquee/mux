# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog] and this project adheres to [Semantic
Versioning].

[Keep a Changelog]: http://keepachangelog.com/en/1.0.0/
[Semantic Versioning]: http://semver.org/spec/v2.0.0.html


## 0.0.4 — 2020–03–19

### Breaking

- Remove `offset` field from [`ParamInfo`] type
- Remove `ok bool` return value from [`Param`] function as it was rarely used
  and is the same as checking `paramInfo.Value != nil`

### Added

- New [`WithParam`] and [`Path`] functions to simplify route normalization

[`ParamInfo`]: https://pkg.go.dev/code.soquee.net/mux#ParamInfo
[`Param`]: https://pkg.go.dev/code.soquee.net/mux#Param
[`WithParam`]: https://pkg.go.dev/code.soquee.net/mux#WithParam
[`Path`]: https://pkg.go.dev/code.soquee.net/mux#Path
