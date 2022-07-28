package httptransport

import (
	"net/http"

	"github.com/dogmatiq/dodeca/logging"
	"github.com/dogmatiq/harpy"
	"go.uber.org/zap"
)

// WithDefaultLogger is a HandlerOption that configures the handler to use a
// harpy.DefaultExchangeLogger for logging requests and responses.
func WithDefaultLogger(logger logging.Logger) HandlerOption {
	return func(h *Handler) {
		h.newLogger = func(r *http.Request) harpy.ExchangeLogger {
			return harpy.DefaultExchangeLogger{
				Target: logging.Prefix(logger, "[%s] ", r.RemoteAddr),
			}
		}
	}
}

// WithZapLogger is a HandlerOption that configures the handler to use a
// harpy.ZapExchangeLogger for logging requests and responses.
func WithZapLogger(logger *zap.Logger) HandlerOption {
	return func(h *Handler) {
		h.newLogger = func(r *http.Request) harpy.ExchangeLogger {
			return harpy.ZapExchangeLogger{
				Target: logger.With(
					zap.String("remote_addr", r.RemoteAddr),
				),
			}
		}
	}
}
