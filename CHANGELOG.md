# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog], and this project adheres to
[Semantic Versioning].

<!-- references -->

[keep a changelog]: https://keepachangelog.com/en/1.0.0/
[semantic versioning]: https://semver.org/spec/v2.0.0.html

## [0.4.0] - 2022-03-25

### Changed

- **[BC]** Changed `Error` to use non-pointer receivers (reversion of change in v0.2.0)
- **[BC]** Added boolean return value to `Request[Set].ValidateServerSide()` and `ValidateClientSide()`

These changes are intended to prevent a subtle and easy to introduce usage
problem that occurs when using a `nil` pointer to a concrete error type to
represent a success, as illustrated below:

```go
// A nil pointer, as was used before this
// version to represent success.
var rpcErr *harpy.Error

// A non-nil interface variable, which
// happens to "contain" a nil pointer.
var err error = rpcErr

if err != nil {
    // This branch executes, even though
    // the original operation was successful
    fmt.Println("an error occurred")
}
```

## [0.3.0] - 2022-03-25

### Added

- Added `NewRouter()`, `WithRoute()` and `NoResult()` to build type-safe routers
- Added `WithUntypedRoute()` to allow continued use of "untyped" handlers with a router

### Changed

- **[BC]** `Router` is now an opaque type (not a map), and uses a pointer receiver
- **[BC]** Renamed `Handler` to `UntypedHandler`

## [0.2.0] - 2022-03-22

### Added

- Added `httptransport.Client`
- Added `NewCallRequest()` and `NewNotifyRequest()`
- Added `Request.ValidateClientSide()` and `RequestSet.ValidateClientSide()`
- Added `ResponseSet` and `UnmarshalResponseSet()`
- Added `Response.Validate()`
- Added `Response.UnmarshalRequestID()`
- Added `Error.MarshalData()` and `UnmarshalData()`
- Added `BatchRequestMarshaler`

### Changed

- **[BC]** Renamed `ParseRequestSet` to `UnmarshalRequestSet` to better match standard library
- **[BC]** Renamed `RequestSet.Validate()` to `ValidateServerSide`
- **[BC]** Renamed `Request.Validate()` to `ValidateServerSide`
- **[BC]** Changed `Error` to use pointer receivers (hence `*Error` now implements `error`)
- `httptransport.Handler` now sends HTTP 204 (no content) in response to a single notification request

### Removed

- **[BC]** Removed `Error.Data()`, use `UnmarshalData()` instead

## [0.1.0] - 2021-08-05

- Initial release

<!-- references -->

[unreleased]: https://github.com/dogmatiq/harpy
[0.1.0]: https://github.com/dogmatiq/harpy/releases/tag/v0.1.0
[0.2.0]: https://github.com/dogmatiq/harpy/releases/tag/v0.2.0
[0.3.0]: https://github.com/dogmatiq/harpy/releases/tag/v0.3.0
[0.4.0]: https://github.com/dogmatiq/harpy/releases/tag/v0.4.0

<!-- version template
## [0.0.1] - YYYY-MM-DD

### Added
### Changed
### Deprecated
### Removed
### Fixed
### Security
-->
