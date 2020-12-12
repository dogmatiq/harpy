package fixtures

import (
	"context"

	"github.com/jmalloc/voorhees"
)

// ExchangerStub is a test implementation of the voorhees.Exchanger interface.
type ExchangerStub struct {
	voorhees.Exchanger

	CallFunc   func(context.Context, voorhees.Request) voorhees.Response
	NotifyFunc func(context.Context, voorhees.Request)
}

// Call handles a call request and returns the response.
func (s *ExchangerStub) Call(ctx context.Context, req voorhees.Request) voorhees.Response {
	if s.CallFunc != nil {
		return s.CallFunc(ctx, req)
	}

	if s.Exchanger != nil {
		return s.Exchanger.Call(ctx, req)
	}

	return nil
}

// Notify handles a notification request.
func (s *ExchangerStub) Notify(ctx context.Context, req voorhees.Request) {
	if s.NotifyFunc != nil {
		s.NotifyFunc(ctx, req)
		return
	}

	if s.Exchanger != nil {
		s.Exchanger.Notify(ctx, req)
	}
}
