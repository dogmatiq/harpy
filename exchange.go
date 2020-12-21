package harpy

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

// A ResponseWriter writes a responses to requests.
type ResponseWriter interface {
	// WriteError writes an error response that is a result of some problem with
	// the request set as a whole.
	WriteError(context.Context, RequestSet, ErrorResponse) error

	// WriteUnbatched writes a response to an individual request that was not
	// part of a batch.
	WriteUnbatched(context.Context, Request, Response) error

	// WriteBatched writes a response to an individual request that was part of
	// a batch.
	WriteBatched(context.Context, Request, Response) error

	// Close is called when there are no more responses to be sent.
	Close() error
}

// Exchange performs a JSON-RPC exchange, whether for a single request or a
// batch of requests.
//
// e is the exchanger used to obtain a response to each request. If there are
// multiple requests each request is passed to the exchanger on its own
// goroutine.
//
// w is used to write responses to each requests. Calls to the methods on w are
// serialized and do not require further synchronization.
//
// If w produces an error, the context passed to e is canceled and Exchange()
// returns the ResponseWriter's error. Execution blocks until all goroutines are
// completed, but no more responses are written.
//
// If ctx is canceled or exceeds its deadline, e is responsible for aborting
// execution and returning a suitable JSON-RPC response describing the
// cancelation. Exchange() does NOT return the context's error.
func Exchange(
	ctx context.Context,
	rs RequestSet,
	e Exchanger,
	w ResponseWriter,
) (err error) {
	defer func() {
		// Always close the responder, but only return its error if there was no
		// more specific error already.
		if e := w.Close(); e != nil {
			if err == nil {
				err = e
			}
		}
	}()

	if err, ok := rs.Validate(); !ok {
		return w.WriteError(
			ctx,
			rs,
			ErrorResponse{
				Version: jsonRPCVersion,
				Error: ErrorInfo{
					Code:    err.Code(),
					Message: err.Message(),
				},
			},
		)
	}

	if rs.IsBatch {
		return exchangeBatch(ctx, rs.Requests, e, w)
	}

	return exchangeSingle(ctx, rs.Requests[0], e, w)
}

// exchangeSingle performs a JSON-RPC exchange for a single request. That is, a
// request that is not part of a batch.
func exchangeSingle(
	ctx context.Context,
	req Request,
	e Exchanger,
	w ResponseWriter,
) error {
	if req.IsNotification() {
		e.Notify(ctx, req)
		return nil
	}

	return w.WriteUnbatched(
		ctx,
		req,
		e.Call(ctx, req),
	)
}

// exchangeBatch performs a JSON-RPC exchange for a batch request.
func exchangeBatch(
	ctx context.Context,
	requests []Request,
	e Exchanger,
	w ResponseWriter,
) error {
	if len(requests) > 1 {
		// If there is actually more than one request then we handle each in its
		// own goroutine.
		return exchangeMany(ctx, requests, e, w)
	}

	// Otherwise we have a batch that happens to contain a single request. We
	// avoid the overhead and latency of starting the extra goroutines and
	// awaiting their completion.
	req := requests[0]

	if req.IsNotification() {
		e.Notify(ctx, req)
		return nil
	}

	return w.WriteBatched(
		ctx,
		req,
		e.Call(ctx, req),
	)
}

// exchangeMany performs an exchange for multiple requests in parallel.
func exchangeMany(
	ctx context.Context,
	requests []Request,
	e Exchanger,
	w ResponseWriter,
) error {
	type exchange struct {
		request  Request
		response Response
	}

	// Create a channel of exchanges on which each response is received. The
	// channel is buffered to ensure that writes do not block even if the
	// Responder panics.
	pending := len(requests)
	exchanges := make(chan exchange, pending)

	// Create a cancelable context so we can abort pending goroutines if the
	// Responder returns an error.
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
			// We only use the Responder if the request is a call and no prior
			// error has occurred.
			err = w.WriteBatched(ctx, x.request, x.response)

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
