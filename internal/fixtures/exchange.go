package fixtures

import (
	"context"

	"github.com/jmalloc/harpy"
)

// ExchangerStub is a test implementation of the Exchanger interface.
type ExchangerStub struct {
	CallFunc   func(context.Context, harpy.Request) harpy.Response
	NotifyFunc func(context.Context, harpy.Request)
}

// Call handles a call request and returns the response.
func (s *ExchangerStub) Call(ctx context.Context, req harpy.Request) harpy.Response {
	if s.CallFunc != nil {
		return s.CallFunc(ctx, req)
	}

	return nil
}

// Notify handles a notification request.
func (s *ExchangerStub) Notify(ctx context.Context, req harpy.Request) {
	if s.NotifyFunc != nil {
		s.NotifyFunc(ctx, req)
	}
}

// ResponseWriterStub is a test implementation of the ResponseWriter interface.
type ResponseWriterStub struct {
	WriteErrorFunc     func(context.Context, harpy.RequestSet, harpy.ErrorResponse) error
	WriteUnbatchedFunc func(context.Context, harpy.Request, harpy.Response) error
	WriteBatchedFunc   func(context.Context, harpy.Request, harpy.Response) error
	CloseFunc          func() error
}

func (s *ResponseWriterStub) WriteError(ctx context.Context, rs harpy.RequestSet, res harpy.ErrorResponse) error {
	if s.WriteErrorFunc != nil {
		return s.WriteErrorFunc(ctx, rs, res)
	}

	return nil
}

func (s *ResponseWriterStub) WriteUnbatched(ctx context.Context, req harpy.Request, res harpy.Response) error {
	if s.WriteUnbatchedFunc != nil {
		return s.WriteUnbatchedFunc(ctx, req, res)
	}

	return nil
}

func (s *ResponseWriterStub) WriteBatched(ctx context.Context, req harpy.Request, res harpy.Response) error {
	if s.WriteBatchedFunc != nil {
		return s.WriteBatchedFunc(ctx, req, res)
	}

	return nil
}

func (s *ResponseWriterStub) Close() error {
	if s.CloseFunc != nil {
		return s.CloseFunc()
	}

	return nil
}