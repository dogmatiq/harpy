package fixtures

import (
	"context"

	"github.com/jmalloc/voorhees"
)

// PipelineStageStub is a test implementation of the voorhees.PipelineStage
// interface.
type PipelineStageStub struct {
	voorhees.PipelineStage

	CallFunc   func(context.Context, voorhees.Request) voorhees.Response
	NotifyFunc func(context.Context, voorhees.Request)
}

// Call handles a call request and returns the response.
func (s *PipelineStageStub) Call(ctx context.Context, req voorhees.Request) voorhees.Response {
	if s.CallFunc != nil {
		return s.CallFunc(ctx, req)
	}

	if s.PipelineStage != nil {
		return s.PipelineStage.Call(ctx, req)
	}

	return nil
}

// Notify handles a notification request.
func (s *PipelineStageStub) Notify(ctx context.Context, req voorhees.Request) {
	if s.NotifyFunc != nil {
		s.NotifyFunc(ctx, req)
		return
	}

	if s.PipelineStage != nil {
		s.PipelineStage.Notify(ctx, req)
	}
}
