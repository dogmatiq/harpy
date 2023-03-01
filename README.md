<div align="center">

# Harpy

A toolkit for writing [JSON-RPC v2.0](https://www.jsonrpc.org/specification)
clients and servers in Go.

[![Documentation](https://img.shields.io/badge/go.dev-documentation-007d9c?&style=for-the-badge)](https://pkg.go.dev/github.com/dogmatiq/harpy)
[![Latest Version](https://img.shields.io/github/tag/dogmatiq/harpy.svg?&style=for-the-badge&label=semver)](https://github.com/dogmatiq/harpy/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/dogmatiq/harpy/ci.yml?style=for-the-badge&branch=main)](https://github.com/dogmatiq/harpy/actions/workflows/ci.yml)
[![Code Coverage](https://img.shields.io/codecov/c/github/dogmatiq/harpy/main.svg?style=for-the-badge)](https://codecov.io/github/dogmatiq/harpy)

</div>

## Example

The [included example](example_test.go) demonstrates how to implement a very
simple in-memory key/value store with a JSON-RPC API.

## Transports

Harpy provides an [HTTP transport](https://pkg.go.dev/github.com/dogmatiq/harpy@main/transport/httptransport)
out of the box, however JSON-RPC 2.0 is a transport-agnostic protocol and as
such Harpy's API attempts to make it easy to implement other transports.
