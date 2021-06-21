package harpy_test

import (
	"context"
	"net/http"
	"sync"

	"github.com/dogmatiq/harpy"
	"github.com/dogmatiq/harpy/transport/httptransport"
)

// Example shows how to implement a very basic JSON-RPC key/value server using
// Harpy's HTTP transport.
func Example() {
	var server KeyValueServer

	// Start the HTTP server.
	http.ListenAndServe(
		":8080",
		&httptransport.Handler{
			Exchanger: harpy.Router{
				"Get": server.Get,
				"Set": server.Set,
			},
		},
	)
}

// KeyValueServer is a very basic key/value store with a JSON-RPC interface.
type KeyValueServer struct {
	// m stores the key/value pairs.
	m sync.Map
}

// Get returns the value associated with a key.
//
// It returns an application-defined error if there is no value associated with
// this key.
func (s *KeyValueServer) Get(_ context.Context, req harpy.Request) (interface{}, error) {
	var params struct {
		Key string `json:"key"`
	}

	if err := req.UnmarshalParameters(&params); err != nil {
		return nil, err
	}

	if value, ok := s.m.Load(params.Key); ok {
		return value, nil
	}

	return nil, harpy.NewError(
		100, // 100 is our example application's error code for "no such key"
		harpy.WithMessage("no such key"),
	)
}

// Set associates a value with a key.
func (s *KeyValueServer) Set(_ context.Context, req harpy.Request) (interface{}, error) {
	var params struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	if err := req.UnmarshalParameters(&params); err != nil {
		return nil, err
	}

	s.m.Store(params.Key, params.Value)

	return nil, nil
}
