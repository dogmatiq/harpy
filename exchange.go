package voorhees

import (
	"context"
)

// An Exchanger produces the responses within an exchange.
//
// An Exchanger differs from a Handler in that an Exchanger is fully responsible
// for producing a valid response to a call, and handling all error conditions.
// It has no facility to return an error.
type Exchanger interface {
	// Call handles call request and returns its response.
	Call(context.Context, Request) Response

	// Notify handels a notification request, which does not expect a response.
	Notify(context.Context, Request)
}

// A BatchResponder is a function that sends a response to one request within a
// batch.
//
// It is NOT used to send singular responses, such as a response to a
// non-batched request or an error response that prevents an entire batch from
// being handled.
type BatchResponder func(req Request, res Response) error

// Exchange performs a JSON-RPC exchange, whether for a single request or a
// batch of requests.
//
// The appropriate method on e is called to handle each request. If there are
// multiple requests each request is handled on its own goroutine.
//
// The response may either be a single response, or a batch of responses. If
// single is true, res is a single response. Note that it's possible for a batch
// request to produce a single response if the response is an error that
// prevented the entire batch from being processed.
//
// If single is false, the response is a batch and r has already been called for
// each response to be sent. Calls to r are serialized and do not require
// further synchronization. r is not called for notification requests.
//
// If r returns an error, the context passed to e is canceled and err is the
// error returned by r. Execution blocks until all goroutines are completed, but
// r is not called again.
//
// If ctx is canceled or exceeds its deadline, e is responsible for aborting
// execution and returning a suitable JSON-RPC response describing the
// cancelation. err is NOT set to the context's error.
func Exchange(
	ctx context.Context,
	rs RequestSet,
	e Exchanger,
	r BatchResponder,
) (res Response, single bool, err error) {
	if err, ok := rs.Validate(); !ok {
		return ErrorResponse{
			Version: jsonRPCVersion,
			Error: ErrorInfo{
				Code:    err.Code(),
				Message: err.Message(),
			},
		}, true, nil
	}

	if rs.IsBatch {
		return exchangeBatch(ctx, rs.Requests, e, r)
	}

	res, ok := exchangeSingle(ctx, rs.Requests[0], e)
	return res, ok, nil
}

// exchangeSingle performs a JSON-RPC exchange for a single request. That is, a
// request that is not part of a batch.
//
// If ok is true, the request is a call and res is the response to that call;
// otherwise, res is nil.
func exchangeSingle(
	ctx context.Context,
	req Request,
	e Exchanger,
) (res Response, ok bool) {
	if req.IsNotification() {
		e.Notify(ctx, req)
		return nil, false
	}

	return e.Call(ctx, req), true
}

// exchangeBatch performs a JSON-RPC exchange for a batch request.
func exchangeBatch(
	ctx context.Context,
	requests []Request,
	e Exchanger,
	r BatchResponder,
) (res Response, single bool, err error) {
	if len(requests) > 1 {
		// If there is actually more than one request then we handle each in its
		// own goroutine.
		return nil, false, exchangeMany(ctx, requests, e, r)
	}

	// Otherwise we have a batch that happens to contain a single request. We
	// avoid the overhead and latency of starting the extra goroutines and
	// waiting their completion.
	req := requests[0]

	if req.IsNotification() {
		e.Notify(ctx, req)
		return nil, false, nil
	}

	return nil, false, r(
		req,
		e.Call(ctx, req),
	)
}

// exchangeMany performs an exchange for multiple requests in parallel.
func exchangeMany(
	ctx context.Context,
	requests []Request,
	e Exchanger,
	r BatchResponder,
) error {
	type exchange struct {
		request  Request
		response Response
	}

	// Create a channel of exchanges on which each response is received. The
	// channel is buffered to ensure that writes do not block even if the
	// BatchResponder panics.
	pending := len(requests)
	exchanges := make(chan exchange, pending)

	// Create a cancelable context so we can abort pending goroutines if the
	// BatchResponder returns an error.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start a goroutine for each request.
	for _, req := range requests {
		x := exchange{request: req}

		go func() {
			if x.request.IsNotification() {
				e.Notify(ctx, x.request)
			} else {
				x.response = e.Call(ctx, x.request)
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
			// We only use the BatchResponder if the request is a call and no
			// prior error has occurred.
			err = r(x.request, x.response)

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
