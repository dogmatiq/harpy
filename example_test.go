package harpy_test

import (
	"context"
	"net/http"
	"sync"

	"github.com/jmalloc/harpy"
	"github.com/jmalloc/harpy/transport/httptransport"
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
func (s *KeyValueServer) Get(_ context.Context, req harpy.Request) (interface{}, error) {
	var params struct {
		Key string `json:"key"`
	}

	if err := req.UnmarshalParameters(&params); err != nil {
		return nil, err
	}

	value, _ := s.m.Load(params.Key)
	return value, nil
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
