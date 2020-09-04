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

// Exchange performs a JSON-RPC exchange, whether a for single request or a
// batch of requests.
//
// The pipeline stage p is called to produce a response for each request.
//
// The response may either be a single response, or a batch of response. If
// single is true, res is a single response.
//
// If single is false, the response is a batch and respond() is called for each
// response to be sent. Calls to respond() are serialized and do not require
// further synchronization. respond() is not called for notification requests.
//
// If respond() returns an error, the context passed to p is canceled and err is
// the error returned by respond(). Execution blocks until all goroutines are
// completed, but respond() is not called again.
//
// If ctx is canceled or exceeds its deadline, p is responsible for aborting
// execution and returning a suitable JSON-RPC response describing the
// cancelation. err is NOT set to the context's error.
func Exchange(
	ctx context.Context,
	rs RequestSet,
	p PipelineStage,
	respond func(Request, Response) error,
) (res Response, single bool, err error) {
	if rs.IsBatch {
		return ExchangeBatch(ctx, rs.Requests, p, respond)
	}

	if len(rs.Requests) != 1 {
		panic("non-batch request sets must contain exactly one request")
	}

	res, ok := ExchangeSingle(ctx, rs.Requests[0], p)
	return res, ok, nil
}

// ExchangeSingle performs a JSON-RPC exchange for a single request. That is, a
// request that is not part of a batch.
//
// The pipeline stage p is called to produce a response.
//
// If ok is true, the request is a call and res is the response to that call.
//
// If ok is false, the request is a notification and res is nil.
//
// If ctx is canceled or exceeds its deadline, p is responsible for aborting
// execution and returning a suitable JSON-RPC response describing the
// cancelation.
func ExchangeSingle(
	ctx context.Context,
	req Request,
	p PipelineStage,
) (res Response, ok bool) {
	if req.IsNotification() {
		p.Notify(ctx, req)
		return nil, false
	}

	return p.Call(ctx, req), true
}

// ExchangeBatch performs a JSON-RPC exchange for a batch request.
//
// The pipeline stage p is called to produce a response for each of the requests
// in the batch.
//
// The response to a batch may either be a single response, or a batch of
// response. If single is true, res is a single response that is relevant to the
// entire batch.
//
// If single is false, the response is a batch and respond() is called for each
// response to be sent. Calls to respond() are serialized and do not require
// further synchronization. respond() is not called for notification requests.
//
// If respond() returns an error, the context passed to p is canceled and err is
// the error returned by respond(). Execution blocks until all goroutines are
// completed, but respond() is not called again.
//
// If ctx is canceled or exceeds its deadline, p is responsible for aborting
// execution and returning a suitable JSON-RPC response describing the
// cancelation. err is NOT set to the context's error.
func ExchangeBatch(
	ctx context.Context,
	requests []Request,
	p PipelineStage,
	respond func(Request, Response) error,
) (res Response, single bool, err error) {
	count := len(requests)

	if count == 0 {
		return ErrorResponse{
			Version: jsonRPCVersion,
			Error: ErrorInfo{
				Code:    InvalidRequestCode,
				Message: "request batches must contain at least one request",
			},
		}, true, nil
	}

	if count > 1 {
		// If there is actually more than one request then we handle each in its
		// own goroutine.
		return nil, false, exchangeMany(ctx, requests, p, respond)
	}

	// Otherwise we have a batch that happens to contain a single request. We
	// avoid the overhead and latency of starting the extra goroutines and
	// waiting their completion.
	req := requests[0]

	if req.IsNotification() {
		p.Notify(ctx, req)
		return nil, false, nil
	}

	return nil, false, respond(
		req,
		p.Call(ctx, req),
	)
}

// exchangeMany performs an exchange for multiple requests in parallel.
func exchangeMany(
	ctx context.Context,
	requests []Request,
	p PipelineStage,
	respond func(Request, Response) error,
) error {
	type exchange struct {
		request  Request
		response Response
	}

	// Create a channel of exchanges on which each response is received. The
	// channel is buffered to ensure that writes do not block even if the
	// user-supplied respond() function panics.
	pending := len(requests)
	exchanges := make(chan exchange, pending)

	// Create a cancelable context so we can abort pending goroutines if any
	// call to respond() returns an error.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start a goroutine for each request.
	for _, req := range requests {
		x := exchange{request: req}

		go func() {
			if x.request.IsNotification() {
				p.Notify(ctx, x.request)
			} else {
				x.response = p.Call(ctx, x.request)
			}

			// We always send the exchange over the channel even for
			// notifications so that we can use it to determine when all
			// goroutines are complete.
			exchanges <- x
		}()
	}

	var err error

	// Wait for each pending goroutine to complete.
	for x := range exchanges {
		if err == nil && !x.request.IsNotification() {
			// We only call respond() if the request is a call and no prior
			// error has occurred.
			err = respond(x.request, x.response)

			if err != nil {
				// We've seen an error for the first time. We cancel the context
				// to abort pending goroutines but continue to wait for them to
				// finish.
				cancel()
			}
		}

		pending--
		if pending == 0 {
			break
		}
	}

	return err
}