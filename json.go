package harpy

import (
	"github.com/dogmatiq/harpy/internal/jsonx"
)

// UnmarshalOption is an option that changes the behavior of JSON unmarshaling.
type UnmarshalOption interface {
	jsonx.UnmarshalOption
}

// AllowUnknownFields is an UnmarshalOption that controls whether parameters,
// results and error data may contain unknown fields.
//
// Unknown fields are disallowed by default.
func AllowUnknownFields(allow bool) interface {
	UnmarshalOption
	RouterOption
} {
	fn := jsonx.UnmarshalOptionFunc(
		func(opts *jsonx.UnmarshalOptionSet) {
			opts.AllowUnknownFields = allow
		},
	)

	return struct {
		jsonx.UnmarshalOptionFunc
		routerOptionFunc
	}{
		fn,
		func(r *Router) {
			r.unmarshalOptions = append(r.unmarshalOptions, fn)
		},
	}
}
