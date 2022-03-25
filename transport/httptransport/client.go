package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync/atomic"

	"github.com/dogmatiq/harpy"
)

// Client is a HTTP-based JSON-RPC client.
type Client struct {
	// HTTPClient is the HTTP client used to make requests. If it is nil,
	// http.DefaultClient is used.
	HTTPClient *http.Client

	// URL is the URL of the JSON-RPC server.
	URL string

	// prevID is the ID of the last "call" request sent. It is incremented by
	// one to generate the next request ID.
	prevID uint32 // atomic
}

// Call invokes a JSON-RPC method.
func (c *Client) Call(
	ctx context.Context,
	method string,
	params, result any,
) error {
	requestID := atomic.AddUint32(&c.prevID, 1)
	req, err := harpy.NewCallRequest(
		requestID,
		method,
		params,
	)
	if err != nil {
		panic(fmt.Sprintf(
			"unable to call JSON-RPC method (%s): %s",
			method,
			err,
		))
	}

	if err, ok := req.ValidateClientSide(); !ok {
		panic(fmt.Sprintf(
			"unable to call JSON-RPC method (%s): %s",
			method,
			err.Message(),
		))
	}

	if !validateResultParameter(result) {
		panic(fmt.Sprintf(
			"unable to call JSON-RPC method (%s): result must be a non-nil pointer",
			method,
		))
	}

	httpRes, err := c.postSingleRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("unable to call JSON-RPC method (%s): %w", method, err)
	}
	defer httpRes.Body.Close()

	res, err := c.unmarshalSingleResponse(httpRes)
	if err != nil {
		return fmt.Errorf("unable to process JSON-RPC response (%s): %w", method, err)
	}

	var requestIDInResponse uint32
	if err := res.UnmarshalRequestID(&requestIDInResponse); err != nil {
		return fmt.Errorf(
			"unable to process JSON-RPC response (%s): request ID in response is expected to be an integer",
			method,
		)
	}

	if requestIDInResponse != requestID {
		return fmt.Errorf(
			"unable to process JSON-RPC response (%s): request ID in response (%d) does not match the actual request ID (%d)",
			method,
			requestIDInResponse,
			requestID,
		)
	}

	switch res := res.(type) {
	case harpy.SuccessResponse:
		if httpRes.StatusCode != http.StatusOK {
			return fmt.Errorf(
				"unable to process JSON-RPC response (%s): unexpected HTTP %d (%s) status code with JSON-RPC success response",
				method,
				httpRes.StatusCode,
				http.StatusText(httpRes.StatusCode),
			)
		}

		if err := json.Unmarshal(res.Result, result); err != nil {
			return fmt.Errorf("unable to process JSON-RPC response (%s): unable to unmarshal result: %w", method, err)
		}

	case harpy.ErrorResponse:
		return harpy.NewClientSideError(
			res.Error.Code,
			res.Error.Message,
			res.Error.Data,
		)
	}

	return nil
}

// Notify sends a JSON-RPC notification.
func (c *Client) Notify(
	ctx context.Context,
	method string,
	params any,
) error {
	req, err := harpy.NewNotifyRequest(
		method,
		params,
	)
	if err != nil {
		panic(fmt.Sprintf(
			"unable to send JSON-RPC notification (%s): %s",
			method,
			err,
		))
	}

	if err, ok := req.ValidateClientSide(); !ok {
		panic(fmt.Sprintf(
			"unable to send JSON-RPC notification (%s): %s",
			method,
			err.Message(),
		))
	}

	httpRes, err := c.postSingleRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("unable to send JSON-RPC notification (%s): %w", method, err)
	}
	defer httpRes.Body.Close()

	// If there is no content that's a "success" as far as a notification is
	// concerned.
	if httpRes.StatusCode == http.StatusNoContent {
		return nil
	}

	// If there is content of any kind, we expect it be a client error,
	// otherwise the server is misbehaving.
	if httpRes.StatusCode < http.StatusBadRequest ||
		httpRes.StatusCode >= http.StatusInternalServerError {
		return fmt.Errorf(
			"unable to process JSON-RPC response (%s): unexpected HTTP %d (%s) status code in response to JSON-RPC notification",
			method,
			httpRes.StatusCode,
			http.StatusText(httpRes.StatusCode),
		)
	}

	res, err := c.unmarshalSingleResponse(httpRes)
	if err != nil {
		return fmt.Errorf("unable to process JSON-RPC response (%s): %w", method, err)
	}

	if res, ok := res.(harpy.ErrorResponse); ok {
		var requestIDInResponse any
		if err := res.UnmarshalRequestID(&requestIDInResponse); err != nil || requestIDInResponse != nil {
			return fmt.Errorf(
				"unable to process JSON-RPC response (%s): request ID in response is expected to be null",
				method,
			)
		}

		return harpy.NewClientSideError(
			res.Error.Code,
			res.Error.Message,
			res.Error.Data,
		)
	}

	// The server has returned a SUCCESSFUL response to a notification, which is
	// nonsensical. Even though this response indicates a success it is likely
	// that a server misbehaving this badly should not be trusted, so we still
	// produce an error.
	return fmt.Errorf(
		"unable to process JSON-RPC response (%s): did not expect a successful JSON-RPC response to a notification, HTTP status code is %d (%s)",
		method,
		httpRes.StatusCode,
		http.StatusText(httpRes.StatusCode),
	)
}

// unmarshalSingleResponse unmarshals a single (non-batched) JSON-RPC response
// from a HTTP response.
func (c *Client) unmarshalSingleResponse(httpRes *http.Response) (harpy.Response, error) {
	if ct := httpRes.Header.Get("Content-Type"); ct != mediaType {
		return nil, fmt.Errorf("unexpected content-type in HTTP response (%s)", ct)
	}

	rs, err := harpy.UnmarshalResponseSet(httpRes.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal JSON-RPC response: %w", err)
	}

	if rs.IsBatch {
		return nil, errors.New("unexpected JSON-RPC batch response")
	}

	return rs.Responses[0], nil
}

// postSingleRequest sends a single (non-batched) request to the server.
func (c *Client) postSingleRequest(
	ctx context.Context,
	req harpy.Request,
) (*http.Response, error) {
	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(req); err != nil {
		// CODE COVERAGE: This should never fail as the request has already been
		// validated.
		panic(err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, body)
	if err != nil {
		// CODE COVERAGE: The main failure case for NewRequestWithContext() is
		// an invalid HTTP method, but we hardcode it here.
		panic(err)
	}

	httpReq.Header.Set("Content-Type", mediaType)

	hc := c.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}

	res, err := hc.Do(httpReq)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// validateResultParameter returns true if r is a valid variable into which a
// JSON-RPC result value can be written.
func validateResultParameter(v any) bool {
	if v == nil {
		return false
	}

	rv := reflect.ValueOf(v)

	if rv.Kind() != reflect.Ptr {
		return false
	}

	if rv.IsNil() {
		return false
	}

	return true
}
