package harpy

import (
	"github.com/dogmatiq/harpy/internal/jsonx"
)

// UnmarshalOption is an option that changes the behavior of JSON unmarshaling.
type UnmarshalOption func(*jsonx.UnmarshalOptions)

// AllowUnknownFields is an UnmarshalOption that controls whether parameters,
// results and error data may contain unknown fields.
//
// Unknown fields are disallowed by default.
func AllowUnknownFields(allow bool) UnmarshalOption {
	return func(opts *jsonx.UnmarshalOptions) {
		opts.AllowUnknownFields = allow
	}
}
