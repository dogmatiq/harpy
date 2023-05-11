# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog], and this project adheres to
[Semantic Versioning].

<!-- references -->

[keep a changelog]: https://keepachangelog.com/en/1.0.0/
[semantic versioning]: https://semver.org/spec/v2.0.0.html

## [0.10.1] - 2023-05-11

### Fixed

- `Router` now calls `Validate()` on call results

## [0.10.0] - 2023-05-11

### Changed

- **[BC]** Add `error` return value to `Exchanger.Notify()`
- **[BC]** Add `error` parameter to `ExchangeLogger.LogNotification()`
- Add `method` attribute to structured logging output

## [0.9.1] - 2023-05-09

### Changed

- **[BC]** Bump Go version requirement from 1.19 to 1.20, see [versioning policy]

## [0.9.0] - 2023-05-09

### Added

- Added `NewZapExchangeLogger()`
- Added `NewSLogExchangeLogger()`

### Removed

- **[BC]** Removed `ZapExchangeLogger` struct

## [0.8.2] - 2023-04-26

### Changed

- Change `WithRoute()` to accept unmarshaling options

## [0.8.1] - 2023-04-25

### Added

- Add `UnmarshalOption` type
- Add `AllowUnknownFields()` option

### Changed

- Change `Request.UnmarshalParameters()` to accept unmarshaling options
- Change `Error.UnmarshalData()` to accept unmarshaling options
- Change `httptransport.Client.Call()` to accept unmarshaling options

## [0.8.0] - 2022-12-02

This release removes Harpy's dependency on the deprecated
`github.com/dogmatiq/dodeca` module. `go.uber.org/zap` is now used as the
default logger.

### Removed

- **[BC]** Remove `harpy.DefaultExchangeLogger`
- **[BC]** Remove `httptransport.WithDefaultLogger()`

## [0.7.0] - 2022-07-29

### Added

- Add `httptransport.NewHandler`
- Add `httptransport.WithDefaultLogger()` and `WithZapLogger()`

### Changed

- **[BC]** All methods on `ExchangeLogger` now require a context
- **[BC]** All fields of `httptransport.Handler` are now unexported

## [0.6.2] - 2022-07-27

### Added

- Add `harpy.ZapExchangeLogger`

## [0.6.1] - 2022-06-21

### Added

- Add `otelharpy.Metrics`, which provides OpenTelemetry metrics for JSON-RPC servers

## [0.6.0] - 2022-06-20

### Changed

- **[BC]** Rename `otelharpy.Tracer` to `Tracing`
- **[BC]** `Tracing` now modifies an the existing (rather than creating a new one) by default

## [0.5.0] - 2022-06-16

### Added

- Added `middleware/otelharpy` package, which provides OpenTelemetry instrumentation of JSON-RPC servers

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

[versioning policy]: https://github.com/dogmatiq/.github/blob/main/VERSIONING.md
[unreleased]: https://github.com/dogmatiq/harpy
[0.1.0]: https://github.com/dogmatiq/harpy/releases/tag/v0.1.0
[0.2.0]: https://github.com/dogmatiq/harpy/releases/tag/v0.2.0
[0.3.0]: https://github.com/dogmatiq/harpy/releases/tag/v0.3.0
[0.4.0]: https://github.com/dogmatiq/harpy/releases/tag/v0.4.0
[0.5.0]: https://github.com/dogmatiq/harpy/releases/tag/v0.5.0
[0.6.0]: https://github.com/dogmatiq/harpy/releases/tag/v0.6.0
[0.6.1]: https://github.com/dogmatiq/harpy/releases/tag/v0.6.1
[0.6.2]: https://github.com/dogmatiq/harpy/releases/tag/v0.6.2
[0.7.0]: https://github.com/dogmatiq/harpy/releases/tag/v0.7.0
[0.8.0]: https://github.com/dogmatiq/harpy/releases/tag/v0.8.0
[0.8.1]: https://github.com/dogmatiq/harpy/releases/tag/v0.8.1
[0.8.2]: https://github.com/dogmatiq/harpy/releases/tag/v0.8.2
[0.9.0]: https://github.com/dogmatiq/harpy/releases/tag/v0.9.0
[0.9.1]: https://github.com/dogmatiq/harpy/releases/tag/v0.9.1
[0.10.0]: https://github.com/dogmatiq/harpy/releases/tag/v0.10.0

<!-- version template
## [0.0.1] - YYYY-MM-DD

### Added
### Changed
### Deprecated
### Removed
### Fixed
### Security
-->
