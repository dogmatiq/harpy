package voorhees

import (
	"context"
	"encoding/json"
	"io"
)

// Serve performs a JSON-RPC exchange by reading the next request (or request
// batch) from r and writing the response(s) to w.
//
// The appropriate method on e is called to handle each request. If there are
// multiple requests each request is handled on its own goroutine.
//
// If ok is false, a malformed request was read from r and an "invalid request"
// response was written w. The request is thus considered to have been handled
// correctly, but r may be in an invalid state such that no future requests can
// be read.
//
// It returns an error if ctx is canceled or an I/O error occurs.
func Serve(
	ctx context.Context,
	e Exchanger,
	r io.Reader,
	w io.Writer,
) (ok bool, err error) {
	rs, err := ParseRequestSet(r)
	if err != nil {
		e, ok := err.(Error)
		if !ok {
			// If ParseRequestSet() returns an error that is NOT an Error, there
			// was some problem reading the request set from r.
			return false, err
		}

		// Otherwise, the request set is maformed in some way. We write a
		// response about the failure and return immediately.
		//
		// At this point, it's possible r is unusable, but only the caller knows
		// if that matters or how to recover so we set ok to false.
		return false, writeResponse(
			w,
			NewErrorResponse(nil, e),
		)
	}

	// nextToken is the next token to write to properly delimit a batch
	// response.
	nextToken := openArray

	res, single, err := Exchange(
		ctx,
		rs,
		e,
		func(
			req Request,
			res Response,
		) error {
			// Write the next token of the batch response.
			if _, err := w.Write(nextToken); err != nil {
				return err
			}

			// We've opened the batch response, so any future responses are
			// separated by a comma instead.
			nextToken = comma

			return writeResponse(w, res)
		},
	)
	if err != nil {
		return false, err
	}

	if single {
		// The response is singular. The respond function above is guaranteed
		// not to have been called, so we now write the single response without
		// any batch delimiting.
		return true, writeResponse(w, res)
	}

	// A batch was written. Batches must always contain at least one request
	// otherwise we would have received a singular error response, so we know we
	// can close the array.
	_, err = w.Write(closeArray)
	return true, err
}

var (
	openArray  = []byte(`[`)
	closeArray = []byte(`]`)
	comma      = []byte(`,`)
)

// writeResponse marshals res to JSON and writes the data to w.
func writeResponse(w io.Writer, res Response) error {
	return json.NewEncoder(w).Encode(res)
}
