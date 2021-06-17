package harpy

import (
	"context"
)

// An Exchanger performs a JSON-RPC exchange, wherein a request is "exchanged"
// for its response.
//
// The Exchanger is responsible for resolving any error conditions. In the case
// of a JSON-RPC call it must also provide the response. It therefore has no
// facility to return an error.
type Exchanger interface {
	// Call handles call request and returns its response.
	Call(context.Context, Request) Response

	// Notify handles a notification request, which does not expect a response.
	Notify(context.Context, Request)
}

// RequestSetReader reads requests sets in order to perform an exchange.
//
// Implementations are typically provided by the transport layer.
type RequestSetReader interface {
	// Read reads the next RequestSet that is to be processed.
	//
	// If there is a problem parsing the request or the request is malformed, an
	// Error is returned. Any other non-nil error should be considered an I/O
	// error. Note that IO error messages are shown to the client.
	Read(ctx context.Context) (RequestSet, error)
}

// A ResponseWriter writes responses to requests.
//
// Implementations are typically provided by the transport layer.
type ResponseWriter interface {
	// WriteError writes an error response that is a result of some problem with
	// the request set as a whole.
	//
	// The given request set is likely invalid or empty.
	WriteError(context.Context, RequestSet, ErrorResponse) error

	// WriteUnbatched writes a response to an individual request that was not
	// part of a batch.
	WriteUnbatched(context.Context, Request, Response) error

	// WriteBatched writes a response to an individual request that was part of
	// a batch.
	WriteBatched(context.Context, Request, Response) error

	// Close is called to signal that there are no more responses to be sent.
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
	e Exchanger,
	r RequestSetReader,
	w ResponseWriter,
	l ExchangeLogger,
) (err error) {
	if l == nil {
		l = DefaultExchangeLogger{}
	}

	defer func() {
		// Always close the responder, but only return its error if there was no
		// more specific error already.
		if e := w.Close(); e != nil {
			if err == nil {
				err = e
			}
		}
	}()

	rs, err := r.Read(ctx)
	if err != nil {
		// As per the RequestSetReader interface, any non-nil error that is NOT
		// an Error is considered an I/O error.
		if _, ok := err.(Error); !ok {
			err = NewErrorWithReservedCode(
				InternalErrorCode,
				WithMessage("unable to read request set: %s", err.Error()),
			)
		}

		res := NewErrorResponse(nil, err)
		l.LogError(rs, res)
		return w.WriteError(ctx, rs, res)
	}

	if err, ok := rs.Validate(); !ok {
		res := ErrorResponse{
			Version: jsonRPCVersion,
			Error: ErrorInfo{
				Code:    err.Code(),
				Message: err.Message(),
			},
		}

		l.LogError(rs, res)
		return w.WriteError(ctx, rs, res)
	}

	if rs.IsBatch {
		return exchangeBatch(ctx, rs.Requests, e, w, l)
	}

	return exchangeSingle(ctx, rs.Requests[0], e, w, l)
}

// exchangeSingle performs a JSON-RPC exchange for a single request. That is, a
// request that is not part of a batch.
func exchangeSingle(
	ctx context.Context,
	req Request,
	e Exchanger,
	w ResponseWriter,
	l ExchangeLogger,
) error {
	if req.IsNotification() {
		e.Notify(ctx, req)
		l.LogNotification(req)
		return nil
	}

	res := e.Call(ctx, req)
	l.LogCall(req, res)

	return w.WriteUnbatched(ctx, req, res)
}

// exchangeBatch performs a JSON-RPC exchange for a batch request.
func exchangeBatch(
	ctx context.Context,
	requests []Request,
	e Exchanger,
	w ResponseWriter,
	l ExchangeLogger,
) error {
	if len(requests) > 1 {
		// If there is actually more than one request then we handle each in its
		// own goroutine.
		return exchangeMany(ctx, requests, e, w, l)
	}

	// Otherwise we have a batch that happens to contain a single request. We
	// avoid the overhead and latency of starting the extra goroutines and
	// awaiting their completion.
	req := requests[0]

	if req.IsNotification() {
		e.Notify(ctx, req)
		l.LogNotification(req)
		return nil
	}

	res := e.Call(ctx, req)
	l.LogCall(req, res)

	return w.WriteBatched(ctx, req, res)
}

// exchangeMany performs an exchange for multiple requests in parallel.
func exchangeMany(
	ctx context.Context,
	requests []Request,
	e Exchanger,
	w ResponseWriter,
	l ExchangeLogger,
) error {
	type exchange struct {
		request  Request
		response Response
	}

	// Create a channel of exchanges on which each response is received. The
	// channel is buffered to ensure that writes do not block even if the
	// ResponseWriter panics.
	pending := len(requests)
	exchanges := make(chan exchange, pending)

	// Create a cancelable context so we can abort pending goroutines if the
	// ResponseWriter returns an error.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start a goroutine for each request.
	for _, req := range requests {
		x := exchange{request: req}

		go func() {
			if x.request.IsNotification() {
				e.Notify(ctx, x.request)
				l.LogNotification(x.request)
			} else {
				x.response = e.Call(ctx, x.request)
				l.LogCall(x.request, x.response)
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
			// We only use the ResponseWriter if the request is a call and no
			// prior error has occurred.
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
