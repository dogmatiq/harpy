package httptransport

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jmalloc/harpy"
)

// responseWriter is an implementation of harpy.ResponseWriter that writes
// responses to an http.ResponseWriter.
type responseWriter struct {
	w         http.ResponseWriter
	enc       *json.Encoder
	arrayOpen bool
}

var (
	openArray  = []byte(`[`)
	closeArray = []byte(`]`)
	comma      = []byte(`,`)
)

// WriteError writes an error response that is a result of some problem with the
// request set as a whole.
//
// It immediately writes the HTTP response headers followed by the HTTP body.
//
// If the error uses one of the error codes reserved by the JSON-RPC
// specification the HTTP status code is set to the most appropriate equivalent.
// Application-defined JSON-RPC errors always result in a HTTP 200 OK, as they
// considered part of normal operation of the transport.
func (w *responseWriter) WriteError(
	_ context.Context,
	_ harpy.RequestSet,
	res harpy.ErrorResponse,
) error {
	return w.writeError(res)
}

// WriteUnbatched writes a response to an individual request that was not part
// of a batch.
//
// It immediately writes the HTTP response headers followed by the HTTP body.
//
// If res is an ErrorResponse and its error code is one of the error codes
// reserved by the JSON-RPC specification the HTTP status code is set to the
// most appropriate equivalent. Application-defined JSON-RPC errors always
// result in a HTTP 200 OK, as they considered part of normal operation of the
// transport.
func (w *responseWriter) WriteUnbatched(
	_ context.Context,
	_ harpy.Request,
	res harpy.Response,
) error {
	if e, ok := res.(harpy.ErrorResponse); ok {
		return w.writeError(e)
	}

	w.w.Header().Set("Content-Type", mediaType)
	w.w.WriteHeader(http.StatusOK)

	return w.enc.Encode(res)
}

// WriteBatched writes a response to an individual request that was part of a
// batch.
//
// If this is the first response of the batch, it immediately writes the HTTP
// response headers and the opening bracket of the array that encapsulates the
// batch of responses.
//
// The HTTP status is always HTTP 200 OK, as even if res is an ErrorResponse,
// other responses in the batch may indicate a success.
func (w *responseWriter) WriteBatched(
	_ context.Context,
	_ harpy.Request,
	res harpy.Response,
) error {
	separator := comma

	if !w.arrayOpen {
		w.w.Header().Set("Content-Type", mediaType)
		w.arrayOpen = true
		separator = openArray
	}

	if _, err := w.w.Write(separator); err != nil {
		return err
	}

	return w.enc.Encode(res)
}

// Close is called to signal that there are no more responses to be sent.
//
// If batched responses have been written, it writes the closing bracket of the
// array that encapsulates the responses.
func (w *responseWriter) Close() error {
	if w.arrayOpen {
		_, err := w.w.Write(closeArray)
		return err
	}

	return nil
}

// writeError sends a JSON-RPC error response using the most appropriate HTTP
// status code.
func (w *responseWriter) writeError(res harpy.ErrorResponse) error {
	status := httpStatusFromErrorCode(res.Error.Code)
	w.w.Header().Set("Content-Type", mediaType)
	w.w.WriteHeader(status)
	return w.enc.Encode(res)
}

// writeErrorWithHTTPStatus writes a JSON-RPC error response using the provided
// HTTP status code.
func (w *responseWriter) writeErrorWithHTTPStatus(status int, res harpy.ErrorResponse) {
	w.w.Header().Set("Content-Type", mediaType)
	w.w.WriteHeader(status)
	w.enc.Encode(res) // nolint:error // no way to report this error to the client, we already failed to write
}

// httpStatusFromErrorCode returns the appropriate HTTP status code to send in
// response to a specific JSON-RPC error code.
//
// Application-defined error codes, that is, error codes that are not reserved
// by the JSON-RPC specification, result in a HTTP status of "200 OK", as they
// are considered part of standard operation of the server.
func httpStatusFromErrorCode(c harpy.ErrorCode) int {
	if !c.IsReserved() {
		return http.StatusOK
	}

	switch c {
	case harpy.ParseErrorCode:
		return http.StatusBadRequest
	case harpy.InvalidRequestCode:
		return http.StatusBadRequest
	case harpy.InvalidParametersCode:
		return http.StatusBadRequest
	case harpy.MethodNotFoundCode:
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}
