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
	params interface{},
	result interface{},
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

	if err := req.ValidateClientSide(); err != nil {
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

	res, err := c.unmarshalSingleResponse(requestID, httpRes)
	if err != nil {
		return fmt.Errorf("unable to process JSON-RPC response (%s): %w", method, err)
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

// unmarshalSingleResponse unmarshals a single (non-batched) JSON-RPC response
// from a HTTP response.
func (c *Client) unmarshalSingleResponse(
	requestID uint32,
	httpRes *http.Response,
) (harpy.Response, error) {
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

	res := rs.Responses[0]
	var requestIDInResponse uint32
	if err := res.UnmarshalRequestID(&requestIDInResponse); err != nil {
		return nil, errors.New("request ID in response is not an integer")
	}

	if requestIDInResponse != requestID {
		return nil, fmt.Errorf(
			"request ID in response (%d) does not match the actual request ID (%d)",
			requestIDInResponse,
			requestID,
		)
	}

	return res, nil
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
		return nil, err
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
func validateResultParameter(v interface{}) bool {
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
