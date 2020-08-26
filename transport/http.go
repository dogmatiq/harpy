package transport

import (
	"net/http"

	"github.com/jmalloc/voorhees"
)

// HTTPHandler is an implementation of http.Handler that decodes JSON-RPC
// requests and passes them to a Handler implementation.
type HTTPHandler struct {
	Handler voorhees.Handler
}

var _ http.Handler = (*HTTPHandler)(nil)

func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
}
