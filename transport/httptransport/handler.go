package httptransport

import (
	"net/http"

	"github.com/dogmatiq/harpy"
)

// mediaType is the MIME media-type for JSON-RPC requests and responses when
// delivered over HTTP.
const mediaType = "application/json"

// Handler is an implementation of http.Handler that provides an HTTP-based
// transport for a JSON-RPC server.
type Handler struct {
	// Exchanger performs JSON-RPC exchanges.
	exchanger harpy.Exchanger

	// newLogger returns the target for log messages about JSON-RPC requests and
	// responses.
	//
	// If it is nil, a harpy.DefaultExchangeLogger is used.
	newLogger func(*http.Request) harpy.ExchangeLogger
}

// HandlerOption configures the behavior of a handler.
type HandlerOption func(*Handler)

// NewHandler returns a new HTTP handler that provides an HTTP-based JSON-RPC
// transport.
func NewHandler(e harpy.Exchanger, options ...HandlerOption) http.Handler {
	h := &Handler{
		exchanger: e,
	}

	WithDefaultLogger(nil)(h)

	for _, opt := range options {
		opt(h)
	}

	return h
}

// ServeHTTP handles the HTTP request.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	harpy.Exchange( // nolint:errcheck // error already logged, nothing more to do
		r.Context(),
		h.exchanger,
		&RequestSetReader{Request: r},
		&ResponseWriter{Target: w},
		h.newLogger(r),
	)
}
