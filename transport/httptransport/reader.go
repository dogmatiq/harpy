package httptransport

import (
	"context"
	"mime"
	"net/http"

	"github.com/dogmatiq/harpy"
)

// RequestSetReader is an implementation of harpy.RequestSetReader that reads a
// JSON-RPC request set from an HTTP request.
type RequestSetReader struct {
	Request *http.Request
}

const (
	// incorrectHTTPMethod is the error message to use when a request is
	// received that does not use the correct HTTP method.
	//
	// This constant is used by the ResponseWriter implementation to send a
	// more-specific HTTP status code when this error occurs.
	incorrectHTTPMethod = "JSON-RPC requests must use the POST method"

	// incorrectMediaType is the error message to use when a request is received
	// that does not use the expected MIME media-type.
	//
	// This constant is used by the ResponseWriter implementation to send a
	// more-specific HTTP status code when this error occurs.
	incorrectMediaType = "JSON-RPC requests must use the application/json content type"
)

// Read reads the next RequestSet that is to be processed.
//
// It returns ctx.Err() if ctx is canceled while waiting to read the next
// request set. If request set data is read but cannot be parsed a native
// JSON-RPC Error is returned. Any other error indicates an IO error.
func (r *RequestSetReader) Read(_ context.Context) (harpy.RequestSet, error) {
	// Check HTTP method is POST.
	if r.Request.Method != http.MethodPost {
		return harpy.RequestSet{}, harpy.NewErrorWithReservedCode(
			harpy.InvalidRequestCode,
			harpy.WithMessage(incorrectHTTPMethod),
		)
	}

	// Validate the "content-type" HTTP header.
	mt, _, err := mime.ParseMediaType(r.Request.Header.Get("Content-Type"))
	if err != nil || mt != mediaType {
		return harpy.RequestSet{}, harpy.NewErrorWithReservedCode(
			harpy.InvalidRequestCode,
			harpy.WithMessage(incorrectMediaType),
		)
	}

	return harpy.UnmarshalRequestSet(r.Request.Body)
}
