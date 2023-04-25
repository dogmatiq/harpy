package jsonx

import (
	"bytes"
	"encoding/json"
	"io"
)

// Decode unmarshals JSON content from r into v.
func Decode[O UnmarshalOption](r io.Reader, v any, options ...O) error {
	var opts UnmarshalOptionSet
	for _, opt := range options {
		opt.applyToUnmarshalOptionSet(&opts)
	}

	dec := json.NewDecoder(r)
	if !opts.AllowUnknownFields {
		dec.DisallowUnknownFields()
	}

	return dec.Decode(&v)
}

// Unmarshal unmarshals JSON content from data into v.
func Unmarshal[O UnmarshalOption](data []byte, v any, options ...O) error {
	return Decode(
		bytes.NewReader(data),
		v,
		options...,
	)
}

// UnmarshalOptionSet is a set of options that control how JSON is unmarshaled.
type UnmarshalOptionSet struct {
	AllowUnknownFields bool
}

// UnmarshalOption is an option that changes the behavior of JSON unmarshaling.
type UnmarshalOption interface {
	applyToUnmarshalOptionSet(*UnmarshalOptionSet)
}

// UnmarshalOptionFunc is a function that implements of UnmarshalOption.
type UnmarshalOptionFunc func(*UnmarshalOptionSet)

func (fn UnmarshalOptionFunc) applyToUnmarshalOptionSet(opts *UnmarshalOptionSet) {
	if fn != nil {
		fn(opts)
	}
}

type UnmarshalOptionAware interface {
	applyUnmarshalOption(UnmarshalOption)
}
