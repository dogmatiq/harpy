package harpy_test

import (
	"context"
	"net/http"
	"sync"

	"github.com/jmalloc/harpy"
)

// ExampleHTTPHandler shows how to implement a very basic JSON-RPC key/value
// server using Harpy's HTTP transport.
func ExampleHTTPHandler() {
	// values contains the key/value pairs stored on our server.
	var values sync.Map

	// getParameters contains the parameters for the "Get" JSON-RPC method.
	type getParameters struct {
		Key string `json:"key"`
	}

	// setParameters represents the parameters for the "Set" JSON-RPC method.
	type setParameters struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	// rpcHandler is a function that handles the JSON-RPC requests that are
	// received by the HTTP server.
	//
	// It conforms the harpy.Handler type.
	rpcHandler := func(ctx context.Context, req harpy.Request) (interface{}, error) {
		switch req.Method {
		case "Get": // The "Get" method retrieves a single key from the store.
			var params getParameters
			if err := req.UnmarshalParameters(&params); err != nil {
				return nil, err
			}

			value, _ := values.Load(params.Key)
			return value, nil

		case "Set": // The "Set" method sets a key to a specific value.
			var params setParameters
			if err := req.UnmarshalParameters(&params); err != nil {
				return nil, err
			}

			values.Store(params.Key, params.Value)
			return nil, nil

		default:
			return nil, harpy.MethodNotFound()
		}
	}

	// Start the HTTP server.
	http.ListenAndServe(
		":8080",
		&harpy.HTTPHandler{
			Exchanger: &harpy.ExchangeLogger{ // Log complete request/response bodies
				Next: &harpy.HandlerInvoker{ // Dispatch to a harpy.Handler
					Handler: rpcHandler,
				},
			},
		},
	)
}
