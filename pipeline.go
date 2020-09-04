package voorhees

import (
	"context"
)

// PipelineStage is a single stage of the JSON-RPC request pipeline.
type PipelineStage interface {
	// Call handles a call request and returns the response.
	Call(context.Context, Request) Response

	// Notify handles a notification request.
	Notify(context.Context, Request)
}
