package httptransport

import (
	"net/http"

	"github.com/dogmatiq/dodeca/logging"
	"github.com/jmalloc/harpy"
)

// mediaType is the MIME media-type for JSON-RPC requests and responses when
// delivered over HTTP.
const mediaType = "application/json"

// Handler is an implementation of http.Handler that provides an HTTP-based
// transport for a JSON-RPC server.
type Handler struct {
	// Exchanger performs JSON-RPC exchanges.
	Exchanger harpy.Exchanger

	// Logger is the target for log messages about JSON-RPC requests and
	// responses.
	Logger logging.Logger
}

// ServeHTTP handles the HTTP request.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := logging.Prefix(h.Logger, "[%s] ", r.RemoteAddr)

	harpy.Exchange( // nolint:errcheck // error already logged, nothing more to do
		r.Context(),
		h.Exchanger,
		&RequestSetReader{Request: r},
		&ResponseWriter{Target: w},
		harpy.DefaultExchangeLogger{Target: logger},
	)
}
