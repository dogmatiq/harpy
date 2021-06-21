package fixtures

import (
	"context"

	"github.com/dogmatiq/harpy"
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

// RequestSetReaderStub is a test implementation of the RequestSetReader
// interface.
type RequestSetReaderStub struct {
	ReadFunc func(context.Context) (harpy.RequestSet, error)
}

func (s *RequestSetReaderStub) Read(ctx context.Context) (harpy.RequestSet, error) {
	if s.ReadFunc != nil {
		return s.ReadFunc(ctx)
	}

	return harpy.RequestSet{}, nil
}

// ResponseWriterStub is a test implementation of the ResponseWriter interface.
type ResponseWriterStub struct {
	WriteErrorFunc     func(harpy.ErrorResponse) error
	WriteUnbatchedFunc func(harpy.Response) error
	WriteBatchedFunc   func(harpy.Response) error
	CloseFunc          func() error
}

func (s *ResponseWriterStub) WriteError(res harpy.ErrorResponse) error {
	if s.WriteErrorFunc != nil {
		return s.WriteErrorFunc(res)
	}

	return nil
}

func (s *ResponseWriterStub) WriteUnbatched(res harpy.Response) error {
	if s.WriteUnbatchedFunc != nil {
		return s.WriteUnbatchedFunc(res)
	}

	return nil
}

func (s *ResponseWriterStub) WriteBatched(res harpy.Response) error {
	if s.WriteBatchedFunc != nil {
		return s.WriteBatchedFunc(res)
	}

	return nil
}

func (s *ResponseWriterStub) Close() error {
	if s.CloseFunc != nil {
		return s.CloseFunc()
	}

	return nil
}
