package httptransport

import (
	"encoding/json"
	"net/http"

	"github.com/dogmatiq/harpy"
)

// ResponseWriter is an implementation of harpy.ResponseWriter that writes
// responses to an http.ResponseWriter.
type ResponseWriter struct {
	// Target is the writer used to send JSON-RPC responses.
	Target http.ResponseWriter

	// arrayOpen indicates whether the JSON opening array bracket has been
	// written as part of a batch response.
	arrayOpen bool
}

var (
	openArray  = []byte(`[`)
	closeArray = []byte(`]`)
	comma      = []byte(`,`)
)

// WriteError writes an error response that is a result of some problem with
// the request set as a whole.
//
// It immediately writes the HTTP response headers followed by the HTTP body.
//
// If the error code is pre-defined by the JSON-RPC specification the HTTP
// status code is set to the most appropriate equivalent, otherwise it is set to
// 500 (Internal Server Error).
func (w *ResponseWriter) WriteError(res harpy.ErrorResponse) error {
	status := httpStatusFromError(res.Error)
	if status == http.StatusOK {
		status = http.StatusInternalServerError
	}

	w.writeHeaders(status)
	return w.writeResponse(res)
}

// WriteUnbatched writes a response to an individual request that was not part
// of a batch.
//
// It immediately writes the HTTP response headers followed by the HTTP body.
//
// If res is an ErrorResponse and its error code is pre-defined by the JSON-RPC
// specification the HTTP status code is set to the most appropriate equivalent.
//
// Application-defined JSON-RPC errors always result in a HTTP 200 (OK), as they
// considered part of normal operation of the transport.
func (w *ResponseWriter) WriteUnbatched(res harpy.Response) error {
	status := http.StatusOK
	if e, ok := res.(harpy.ErrorResponse); ok {
		status = httpStatusFromError(e.Error)
	}

	w.writeHeaders(status)
	return w.writeResponse(res)
}

// WriteBatched writes a response to an individual request that was part of a
// batch.
//
// If this is the first response of the batch, it immediately writes the HTTP
// response headers and the opening bracket of the array that encapsulates the
// batch of responses.
//
// The HTTP status code is always 200 (OK), as even if res is an ErrorResponse,
// other responses in the batch may indicate a success.
func (w *ResponseWriter) WriteBatched(res harpy.Response) error {
	separator := comma

	if !w.arrayOpen {
		w.writeHeaders(http.StatusOK)
		w.arrayOpen = true
		separator = openArray
	}

	if _, err := w.Target.Write(separator); err != nil {
		return err
	}

	return w.writeResponse(res)
}

// Close is called to signal that there are no more responses to be sent.
//
// If batched responses have been written, it writes the closing bracket of the
// array that encapsulates the responses.
func (w *ResponseWriter) Close() error {
	if w.arrayOpen {
		_, err := w.Target.Write(closeArray)
		return err
	}

	return nil
}

// writeHeaders writes the HTTP response headers.
func (w *ResponseWriter) writeHeaders(status int) {
	w.Target.Header().Set("Content-Type", mediaType)
	w.Target.WriteHeader(status)
}

// writeResponse writes a JSON-RPC response to the HTTP response body.
func (w *ResponseWriter) writeResponse(res harpy.Response) error {
	enc := json.NewEncoder(w.Target)
	return enc.Encode(res)
}

// httpStatusFromError returns the appropriate HTTP status code to send in
// response to a specific JSON-RPC error code.
//
// Application-defined error codes, that is, error codes that are not reserved
// by the JSON-RPC specification, result in a HTTP status of "200 OK", as they
// are considered part of standard operation of the server.
func httpStatusFromError(err harpy.ErrorInfo) int {
	if !err.Code.IsReserved() {
		return http.StatusOK
	}

	switch err.Code {
	case harpy.ParseErrorCode:
		return http.StatusBadRequest

	case harpy.InvalidRequestCode:
		// Return more specific HTTP status codes for the "invalid request"
		// errors that were produced by the HTTP-specific RequestSetReader from
		// this package.
		if err.Message == incorrectHTTPMethod {
			return http.StatusMethodNotAllowed
		} else if err.Message == incorrectMediaType {
			return http.StatusUnsupportedMediaType
		}

		return http.StatusBadRequest

	case harpy.InvalidParametersCode:
		return http.StatusBadRequest

	case harpy.MethodNotFoundCode:
		return http.StatusNotImplemented

	default:
		return http.StatusInternalServerError
	}
}
