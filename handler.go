package voorhees

import "context"

// Handler handles JSON-RPC requests and notifications.
type Handler interface {
	Call(context.Context, Request) (interface{}, error)
	Notify(context.Context, Request) error
}
