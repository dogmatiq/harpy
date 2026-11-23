<div align="center">

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://dogmatiq.io/res/img/dynamic/repo/harpy/banner/dark.svg">
  <img alt="Dogmatiq harpy" src="https://dogmatiq.io/res/img/dynamic/repo/harpy/banner/light.svg" height="44">
</picture>

A toolkit for writing [JSON-RPC v2.0](https://www.jsonrpc.org/specification)
clients and servers in Go.

[![API reference](https://dogmatiq.io/res/img/dynamic/repo/harpy/badge/docs.svg)](https://pkg.go.dev/github.com/dogmatiq/harpy)
[![Latest version](https://dogmatiq.io/res/img/dynamic/repo/harpy/badge/semver.svg)](https://github.com/dogmatiq/harpy/releases)
[![Build status](https://dogmatiq.io/res/img/dynamic/repo/harpy/badge/build.svg)](https://github.com/dogmatiq/harpy/actions/workflows/ci.yml)
[![Code coverage](https://dogmatiq.io/res/img/dynamic/repo/harpy/badge/coverage.svg)](https://codecov.io/github/dogmatiq/harpy)

</div>

## Example

The [included example](example_test.go) demonstrates how to implement a very
simple in-memory key/value store with a JSON-RPC API.

## Transports

Harpy provides an [HTTP transport](https://pkg.go.dev/github.com/dogmatiq/harpy@main/transport/httptransport)
out of the box, however JSON-RPC 2.0 is a transport-agnostic protocol and as
such Harpy's API attempts to make it easy to implement other transports.
