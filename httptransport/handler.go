package httptransport

import (
	"errors"
	"fmt"
	"mime"
	"net/http"

	"github.com/jmalloc/harpy"
)

// Handler is an implementation of http.Handler that provides an HTTP-based
// transport for a JSON-RPC server.
type Handler struct {
	// Exchanger performs JSON-RPC exchanges.
	Exchanger harpy.Exchanger
}

// ServeHTTP handles the HTTP request.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rw := &ResponseWriter{
		Target: w,
	}

	if !validateHeaders(rw, r) {
		return
	}

	rs, ok := parseRequestSet(rw, r)
	if !ok {
		return
	}

	// Perform the exchange. Any error here is an IO problem with the HTTP
	// response, so we can't inform the HTTP client about it in any way.
	//
	// We leave it up to hypotethetical HTTP middleware to log the error, if
	// necessary.
	harpy.Exchange( // nolint:errcheck
		r.Context(),
		rs,
		h.Exchanger,
		rw,
	)
}

// mediaType is the MIME media-type for JSON-RPC requests and responses when
// delivered over HTTP.
const mediaType = "application/json"

// validateHeaders checks that the necessary HTTP request headers are set
// correctly.
//
// If any header values are invalid it writes a JSON-RPC error to rw and returns
// false.
func validateHeaders(rw *ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		rw.writeErrorWithHTTPStatus(
			http.StatusMethodNotAllowed,
			harpy.NewErrorResponse(
				nil,
				harpy.NewErrorWithReservedCode(
					harpy.InvalidRequestCode,
					harpy.WithMessage("JSON-RPC requests must use the POST method"),
				),
			),
		)

		return false
	}

	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != mediaType {
		rw.writeErrorWithHTTPStatus(
			http.StatusUnsupportedMediaType,
			harpy.NewErrorResponse(
				nil,
				harpy.NewErrorWithReservedCode(
					harpy.InvalidRequestCode,
					harpy.WithMessage("JSON-RPC requests must use the application/json content type"),
				),
			),
		)

		return false
	}

	return true
}

// parseRequestSet parses a JSON-RPC request set from a HTTP request.
//
// If parsing fails it writes a JSON-RPC error to rw and sets ok to false.
func parseRequestSet(rw *ResponseWriter, r *http.Request) (_ harpy.RequestSet, ok bool) {
	rs, err := harpy.ParseRequestSet(r.Body)
	if err == nil {
		return rs, true
	}

	// There was a problem with the JSON-RPC request set.
	var jsonErr harpy.Error
	if errors.As(err, &jsonErr) {
		res := harpy.NewErrorResponse(nil, err)
		rw.writeError(res) // nolint:errcheck // no way to report this error to the client, we already failed to write
		return harpy.RequestSet{}, false
	}

	// Otherwise, there was a problem reading the HTTP request.
	rw.writeErrorWithHTTPStatus(
		http.StatusInternalServerError,
		harpy.NewErrorResponse(
			nil,
			harpy.NewErrorWithReservedCode(
				harpy.InternalErrorCode,
				harpy.WithCause(
					fmt.Errorf("unable to read HTTP request body: %w", err),
				),
			),
		),
	)

	return harpy.RequestSet{}, false
}
