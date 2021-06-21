# Harpy

[![Build Status](https://github.com/dogmatiq/harpy/workflows/CI/badge.svg)](https://github.com/dogmatiq/harpy/actions?workflow=CI)
[![Code Coverage](https://img.shields.io/codecov/c/github/dogmatiq/harpy/main.svg)](https://codecov.io/github/dogmatiq/harpy)
[![Latest Version](https://img.shields.io/github/tag/dogmatiq/harpy.svg?label=semver)](https://semver.org)
[![Documentation](https://img.shields.io/badge/go.dev-reference-007d9c)](https://pkg.go.dev/github.com/dogmatiq/harpy)
[![Go Report Card](https://goreportcard.com/badge/github.com/dogmatiq/harpy)](https://goreportcard.com/report/github.com/dogmatiq/harpy)

Harpy is a toolkit for writing [JSON-RPC v2.0](https://www.jsonrpc.org/specification)
servers with Go.

## Transports

Harpy provides an [HTTP transport](https://pkg.go.dev/github.com/dogmatiq/harpy@main/transport/httptransport)
out of the box, however JSON-RPC 2.0 is a transport-agnostic protocol and as
such Harpy's API attempts to make it easy to implement other transports.

## Example Server

The [included example](example_test.go) demonstrates how to implement a
very simple in-memory key/value store with a JSON-RPC API.
