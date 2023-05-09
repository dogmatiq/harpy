package harpy

import (
	"context"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
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
	// It returns ctx.Err() if ctx is canceled while waiting to read the next
	// request set. If request set data is read but cannot be parsed a native
	// JSON-RPC Error is returned. Any other error indicates an IO error.
	Read(ctx context.Context) (RequestSet, error)
}

// A ResponseWriter writes responses to requests.
//
// Implementations are typically provided by the transport layer.
type ResponseWriter interface {
	// WriteError writes an error response that is a result of some problem with
	// the request set as a whole.
	WriteError(ErrorResponse) error

	// WriteUnbatched writes a response to an individual request that was not
	// part of a batch.
	WriteUnbatched(Response) error

	// WriteBatched writes a response to an individual request that was part of
	// a batch.
	WriteBatched(Response) error

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
// r is used to obtain a the next RequestSet to process, and w is used to write
// responses to each request in that set. Calls to the methods on w are
// serialized and do not require further synchronization.
//
// If ctx is canceled or exceeds its deadline, e is responsible for aborting
// execution and returning a suitable JSON-RPC response describing the
// cancelation.
//
// If w produces an error, the context passed to e is canceled and Exchange()
// returns the ResponseWriter's error. Execution blocks until all goroutines are
// completed, but no more responses are written.
func Exchange(
	ctx context.Context,
	e Exchanger,
	r RequestSetReader,
	w ResponseWriter,
	l ExchangeLogger,
) (err error) {
	if l == nil {
		t, err := zap.NewProduction()
		if err != nil {
			return err
		}

		l = NewZapExchangeLogger(t)
	}

	defer func() {
		// Always close the writer, but only return its error if there was no
		// more specific error already.
		if e := w.Close(); e != nil {
			l.LogWriterError(ctx, e)

			if err == nil {
				err = e
			}
		}
	}()

	rs, ok, err := readRequestSet(ctx, r, w, l)
	if !ok || err != nil {
		return err
	}

	if rs.IsBatch {
		return exchangeBatch(ctx, e, rs.Requests, w, l)
	}

	return exchangeSingle(ctx, e, rs.Requests[0], w, l)
}

// readRequestSet returns the next request set from r.
//
// It returns an error if ctx has been canceled or an IO error occurs such that
// Exchange() should return to the caller.
//
// Otherwise; if ok is true the request set is valid and needs to be processed.
// If ok is false, there was some other problem with the request set that has
// already been reported to the client.
func readRequestSet(
	ctx context.Context,
	r RequestSetReader,
	w ResponseWriter,
	l ExchangeLogger,
) (_ RequestSet, ok bool, _ error) {
	rs, readErr := r.Read(ctx)
	if readErr != nil {
		if readErr == ctx.Err() {
			// The context was canceled while waiting for the next request set,
			// return the error to the caller without doing anything. The would
			// be the typical path used to abort execution of a blocked call to
			// Exchange().
			return RequestSet{}, false, readErr
		}

		if _, ok := readErr.(Error); ok {
			// There was no problem reading data for the request set, but it
			// could not be parsed as JSON.
			res := NewErrorResponse(nil, readErr)
			l.LogError(ctx, res)

			if writeErr := w.WriteError(res); writeErr != nil {
				l.LogWriterError(ctx, writeErr)
				return RequestSet{}, false, writeErr
			}

			return RequestSet{}, false, nil
		}

		// Otherwise; any non-nil error is an IO error. We still try to report
		// something meaningful to the client, but it's likely that if reading
		// failed that writing will also fail.
		res := NewErrorResponse(
			nil,
			NewErrorWithReservedCode(
				InternalErrorCode,
				WithMessage("unable to read JSON-RPC request"),
				WithCause(readErr),
			),
		)
		l.LogError(ctx, res)

		if writeErr := w.WriteError(res); writeErr != nil {
			l.LogWriterError(ctx, writeErr)
			// Don't return the writeErr, preferring instead to return the
			// readErr that happened first.
		}

		return RequestSet{}, false, readErr
	}

	if err, ok := rs.ValidateServerSide(); !ok {
		// The request data is well-formed JSON but not a valid JSON-RPC request
		// or batch.
		res := newNativeErrorResponse(nil, err)
		l.LogError(ctx, res)

		if writeErr := w.WriteError(res); writeErr != nil {
			l.LogWriterError(ctx, writeErr)
			return RequestSet{}, false, writeErr
		}

		return RequestSet{}, false, nil
	}

	return rs, true, nil
}

// exchangeOne performs a JSON-RPC exchange for one request and writes the
// response using w.
func exchangeOne(
	ctx context.Context,
	e Exchanger,
	req Request,
	w func(Response) error,
	l ExchangeLogger,
) error {
	if req.IsNotification() {
		e.Notify(ctx, req)
		l.LogNotification(ctx, req)
		return nil
	}

	res := e.Call(ctx, req)
	l.LogCall(ctx, req, res)

	if err := w(res); err != nil {
		l.LogWriterError(ctx, err)
		return err
	}

	return nil
}

// exchangeSingle performs a JSON-RPC exchange for a single (non-batched)
// request.
func exchangeSingle(
	ctx context.Context,
	e Exchanger,
	req Request,
	w ResponseWriter,
	l ExchangeLogger,
) error {
	return exchangeOne(
		ctx,
		e,
		req,
		w.WriteUnbatched,
		l,
	)
}

// exchangeBatch performs a JSON-RPC exchange for a batch of requests.
func exchangeBatch(
	ctx context.Context,
	e Exchanger,
	requests []Request,
	w ResponseWriter,
	l ExchangeLogger,
) error {
	if len(requests) > 1 {
		// If there is actually more than one request then we handle each in its
		// own goroutine.
		return exchangeMany(ctx, e, requests, w, l)
	}

	// Otherwise we have a batch that happens to contain a single request. We
	// avoid the overhead and latency of starting the extra goroutines and
	// awaiting their completion.
	return exchangeOne(
		ctx,
		e,
		requests[0],
		w.WriteBatched,
		l,
	)
}

// exchangeMany performs an exchange for multiple requests in parallel.
func exchangeMany(
	ctx context.Context,
	e Exchanger,
	requests []Request,
	w ResponseWriter,
	l ExchangeLogger,
) error {

	var (
		m  sync.Mutex // synchronise access to w and ok
		ok = true
	)

	// Create an errgroup to abort any pending calls to the exchanger if an
	// error occurs when writing responses.
	g, ctx := errgroup.WithContext(ctx)

	// Start a goroutine for each request.
	for _, req := range requests {
		req := req // capture loop variable

		g.Go(func() error {
			return exchangeOne(
				ctx,
				e,
				req,
				func(res Response) error {
					m.Lock()
					defer m.Unlock()

					// Only write the response if there has not already been
					// an error writing responses.
					if ok {
						err := w.WriteBatched(res)
						ok = err == nil
						return err
					}

					return nil
				},
				l,
			)
		})
	}

	return g.Wait()
}
