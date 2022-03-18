# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog], and this project adheres to
[Semantic Versioning].

<!-- references -->

[keep a changelog]: https://keepachangelog.com/en/1.0.0/
[semantic versioning]: https://semver.org/spec/v2.0.0.html

## [Unreleased]

### Added

- Added `JSONRPCVersion` constant
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

<!-- version template
## [0.0.1] - YYYY-MM-DD

### Added
### Changed
### Deprecated
### Removed
### Fixed
### Security
-->
