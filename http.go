package voorhees

import (
	"context"
	"encoding/json"
	"mime"
	"net/http"
)

// HTTPHandler is an implementation of http.Handler that provides an HTTP-based
// transport for a JSON-RPC server.
type HTTPHandler struct {
	// Exchanger handles JSON-RPC requests.
	Exchanger Exchanger
}

// ServeHTTP handles the HTTP request.
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rw := &httpResponseWriter{
		w:   w,
		enc: json.NewEncoder(w),
	}

	if r.Method != http.MethodPost {
		rw.writeError(
			http.StatusMethodNotAllowed,
			NewErrorResponse(
				nil,
				NewErrorWithReservedCode(
					InvalidRequestCode,
					WithMessage("JSON-RPC requests must use the POST method"),
				),
			),
		)
		return
	}

	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != httpMediaType {
		rw.writeError(
			http.StatusUnsupportedMediaType,
			NewErrorResponse(
				nil,
				NewErrorWithReservedCode(
					InvalidRequestCode,
					WithMessage("JSON-RPC requests must use the application/json content type"),
				),
			),
		)
		return
	}

	ctx := r.Context()

	rs, err := ParseRequestSet(r.Body)

	switch err.(type) {
	case nil:
		err = Exchange(
			ctx,
			rs,
			h.Exchanger,
			rw,
		)
	case Error:
		err = rw.WriteError(
			ctx,
			RequestSet{},
			NewErrorResponse(nil, err),
		)
	default:
		rw.writeError(
			http.StatusInternalServerError,
			NewErrorResponse(
				nil,
				NewErrorWithReservedCode(
					InternalErrorCode,
					WithMessage("unable to read request body"),
				),
			),
		)
		return
	}
}

const httpMediaType = "application/json"

var (
	openArray  = []byte(`[`)
	closeArray = []byte(`]`)
	comma      = []byte(`,`)
)

// httpResponseWriter is an implementation of ResponseWriter that sends
// responses to HTTP requests.
type httpResponseWriter struct {
	w       http.ResponseWriter
	enc     *json.Encoder
	isBatch bool
}

func (w *httpResponseWriter) WriteError(_ context.Context, _ RequestSet, res ErrorResponse) error {
	return w.writeError(0, res)
}

func (w *httpResponseWriter) WriteUnbatched(_ context.Context, _ Request, res Response) error {
	if e, ok := res.(ErrorResponse); ok {
		return w.writeError(0, e)
	}

	w.w.Header().Set("Content-Type", httpMediaType)
	return w.enc.Encode(res)
}

func (w *httpResponseWriter) WriteBatched(_ context.Context, _ Request, res Response) error {
	separator := comma

	if !w.isBatch {
		w.w.Header().Set("Content-Type", httpMediaType)
		w.isBatch = true
		separator = openArray
	}

	if _, err := w.w.Write(separator); err != nil {
		return err
	}

	return w.enc.Encode(res)
}

func (w *httpResponseWriter) Close() error {
	if w.isBatch {
		_, err := w.w.Write(closeArray)
		return err
	}

	return nil
}

func (w *httpResponseWriter) writeError(code int, res ErrorResponse) error {
	if code == 0 {
		code = httpStatusFromErrorCode(res.Error.Code)
	}

	w.w.Header().Set("Content-Type", httpMediaType)
	w.w.WriteHeader(code)
	return w.enc.Encode(res)
}

// httpStatusFromErrorCode returns the appropriate HTTP status code to send in
// response to a specific JSON-RPC error code.
func httpStatusFromErrorCode(c ErrorCode) int {
	if !c.IsReserved() {
		// If the error code is not "reserved" that means its an
		// application-defined error. We do write the response using an OK
		// status as even though an error occurred there was no problem with the
		// request or the HTTP encapsulation itself.
		return http.StatusOK
	}

	switch c {
	case ParseErrorCode:
		return http.StatusBadRequest
	case InvalidRequestCode:
		return http.StatusBadRequest
	case InvalidParametersCode:
		return http.StatusBadRequest
	case MethodNotFoundCode:
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}
