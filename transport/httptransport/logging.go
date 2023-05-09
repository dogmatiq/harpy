package httptransport

import (
	"net/http"

	"github.com/dogmatiq/harpy"
	"go.uber.org/zap"
)

// WithZapLogger is a HandlerOption that configures the handler to use a
// harpy.ZapExchangeLogger for logging requests and responses.
func WithZapLogger(logger *zap.Logger) HandlerOption {
	return func(h *Handler) {
		h.newLogger = func(r *http.Request) harpy.ExchangeLogger {
			return harpy.NewZapExchangeLogger(
				logger.With(
					zap.String("remote_addr", r.RemoteAddr),
				),
			)
		}
	}
}
