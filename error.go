package harpy

import (
	"encoding/json"
	"fmt"
	"sync"
)

// Error is a Go error that describes a JSON-RPC error.
type Error struct {
	// code is the JSON-RPC error code.
	code ErrorCode

	// message is the user-defined error message, if any.
	message string

	// data is the user-defined error value that is attached to the error, if
	// any.
	data errorData

	// isServerSide indicates whether or not this error was created within a
	// Harpy JSON-RPC server and is intended to be delivered to the caller.
	//
	// If it is false, this error was returned by a JSON-RPC client.
	isServerSide bool

	// cause is the Go error that caused this JSON-RPC error, if any.
	cause error
}

// newError returns a new Error with the given code.
//
// The options are applied in order.
func newError(code ErrorCode, options []ErrorOption) Error {
	e := Error{
		code:         code,
		isServerSide: true,
	}

	for _, opt := range options {
		opt(&e)
	}

	return e
}

// NewError returns a new JSON-RPC error with an application-defined error code.
//
// The error codes from and including -32768 to -32000 are reserved for
// pre-defined errors by the JSON-RPC specification. Use of a code within this
// range causes a panic.
func NewError(code ErrorCode, options ...ErrorOption) Error {
	if code.IsReserved() {
		panic(fmt.Sprintf("the error code %d is reserved by the JSON-RPC specification (%s)", code, code))
	}

	return newError(code, options)
}

// NewErrorWithReservedCode returns a new JSON-RPC error that uses a reserved
// error code.
//
// The error codes from and including -32768 to -32000 are reserved for
// pre-defined errors by the JSON-RPC specification. Use of a code outside this
// range causes a panic.
//
// This function is provided to allow user-defined handlers to produce errors
// with reserved codes if necessary, but forces the developer to be explicit
// about doing so.
//
// Consider using MethodNotFound(), InvalidParameters() or NewError() instead.
func NewErrorWithReservedCode(code ErrorCode, options ...ErrorOption) Error {
	if !code.IsReserved() {
		panic(fmt.Sprintf("the error code %d is not reserved by the JSON-RPC specification", code))
	}

	return newError(code, options)
}

// NewClientSideError returns a new client-side error that represents a JSON-RPC
// error returned as part of an ErrorResponse.
func NewClientSideError(
	code ErrorCode,
	message string,
	data json.RawMessage,
) Error {
	err := Error{
		code:         code,
		message:      message,
		isServerSide: false,
	}

	if data != nil {
		err.data = jsonErrorData(data)
	}

	return err
}

// MethodNotFound returns an error that indicates the requested method does not
// exist.
func MethodNotFound(options ...ErrorOption) Error {
	return newError(MethodNotFoundCode, options)
}

// InvalidParameters returns an error that indicates the provided parameters are
// malformed or invalid.
func InvalidParameters(options ...ErrorOption) Error {
	return newError(InvalidParametersCode, options)
}

// Code returns the JSON-RPC error code.
func (e Error) Code() ErrorCode {
	return e.code
}

// Message returns the error message.
func (e Error) Message() string {
	if e.message != "" {
		return e.message
	}

	return e.code.String()
}

// MarshalData returns the JSON representation user-defined data value
// associated with the error.
//
// ok is false if there is no user-defined data associated with the error.
func (e Error) MarshalData() (_ json.RawMessage, ok bool, _ error) {
	if e.data == nil {
		return nil, false, nil
	}

	data, err := e.data.Marshal()
	return data, true, err
}

// UnmarshalData unmarshals the user-defined data into v.
//
// ok is false if there is no user-defined data associated with the error.
func (e Error) UnmarshalData(v any) (ok bool, _ error) {
	data, ok, err := e.MarshalData()
	if !ok || err != nil {
		return false, err
	}

	return true, json.Unmarshal(data, v)
}

// Error returns the error message.
func (e Error) Error() string {
	return describeError(e.code, e.message)
}

// Unwrap returns the cause of e, if known.
func (e Error) Unwrap() error {
	return e.cause
}

// ErrorOption is an option that provides further information about an error.
type ErrorOption func(*Error)

// WithCause is an ErrorOption that associates a causal error with a JSON-RPC
// error.
//
// c is wrapped by the resulting JSON-RPC error, such as it can be used with
// errors.Is() and errors.As().
//
// If the JSON-RPC error does not already have a user-defined message, c.Error()
// is used as the user-defined message.
func WithCause(c error) ErrorOption {
	return func(e *Error) {
		e.cause = c

		if e.message == "" {
			// If there is no user-defined error message already provided, use
			// this error as the message.
			e.message = c.Error()
		}
	}
}

// WithMessage is an ErrorOption that provides a user-defined error message for
// a JSON-RPC error.
//
// This message should be used to provide additional information that can help
// diagnose the error.
func WithMessage(format string, values ...any) ErrorOption {
	return func(e *Error) {
		e.message = fmt.Sprintf(format, values...)
	}
}

// WithData is an ErrorOption that associates additional data with an error.
//
// The data is provided to the RPC caller via the "data" field of the error
// object in the JSON-RPC response.
func WithData(data any) ErrorOption {
	return func(e *Error) {
		e.data = &inMemoryErrorData{value: data}
	}
}

// errorData is an interface for user-defined error data values.
type errorData interface {
	Marshal() (json.RawMessage, error)
}

// jsonErrorData is an implementation of errorData that contains a pre-marshaled
// JSON representation of the error data.
type jsonErrorData json.RawMessage

func (e jsonErrorData) Marshal() (json.RawMessage, error) {
	return json.RawMessage(e), nil
}

// inMemoryErrorData is an implementation of errorData that contains an
// in-memory Go value representation of the error data.
type inMemoryErrorData struct {
	value any

	once sync.Once
	data json.RawMessage
	err  error
}

func (e *inMemoryErrorData) Marshal() (json.RawMessage, error) {
	e.once.Do(func() {
		e.data, e.err = json.Marshal(e.value)
	})

	return e.data, e.err
}
