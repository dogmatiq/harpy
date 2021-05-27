package harpy_test

import (
	"context"
	"net/http"
	"sync"

	"github.com/jmalloc/harpy"
	"github.com/jmalloc/harpy/httptransport"
)

// Example shows how to implement a very basic JSON-RPC key/value server using
// Harpy's HTTP transport.
func Example() {
	var server KeyValueServer

	// Start the HTTP server.
	http.ListenAndServe(
		":8080",
		&httptransport.Handler{ // Use HTTP to deliver requests/responses
			Exchanger: &harpy.ExchangeLogger{ // Log complete request/response bodies
				Next: &harpy.HandlerInvoker{ // Dispatch to a harpy.Handler
					Handler: server.Handle, // Use the key/value server as the handler
				},
			},
		},
	)
}

// KeyValueServer is a very basic key/value store with a JSON-RPC interface.
type KeyValueServer struct {
	// m stores the key/value pairs.
	m sync.Map
}

// Handle handles a JSON-RPC request.
//
// It conforms the harpy.Handler type.
func (s *KeyValueServer) Handle(ctx context.Context, req harpy.Request) (interface{}, error) {
	switch req.Method {
	case "Get":
		return s.get(req)
	case "Set":
		return nil, s.set(req)
	default:
		return nil, harpy.MethodNotFound()
	}
}

// get returns the value associated with a key.
func (s *KeyValueServer) get(req harpy.Request) (interface{}, error) {
	var params struct {
		Key string `json:"key"`
	}

	if err := req.UnmarshalParameters(&params); err != nil {
		return nil, err
	}

	value, _ := s.m.Load(params.Key)
	return value, nil
}

// set associates a value with a key.
func (s *KeyValueServer) set(req harpy.Request) error {
	var params struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	if err := req.UnmarshalParameters(&params); err != nil {
		return err
	}

	s.m.Store(params.Key, params.Value)
	return nil
}
