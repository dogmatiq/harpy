package jsonx

import (
	"bytes"
	"encoding/json"
	"io"
)

// Decode unmarshals JSON content from r into v.
func Decode[O ~UnmarshalOption](
	r io.Reader,
	v any,
	options ...O,
) error {
	var opts UnmarshalOptions
	for _, fn := range options {
		fn(&opts)
	}

	dec := json.NewDecoder(r)
	if !opts.AllowUnknownFields {
		dec.DisallowUnknownFields()
	}

	return dec.Decode(&v)
}

// Unmarshal unmarshals JSON content from data into v.
func Unmarshal[O ~UnmarshalOption](
	data []byte,
	v any,
	options ...O,
) error {
	return Decode(
		bytes.NewReader(data),
		v,
		options...,
	)
}

// UnmarshalOptions is a set of options that control how JSON is unmarshaled.
type UnmarshalOptions struct {
	AllowUnknownFields bool
}

// UnmarshalOption is a function that changes the behavior of JSON unmarshaling.
type UnmarshalOption = func(*UnmarshalOptions)
