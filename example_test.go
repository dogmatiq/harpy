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
			Exchanger: harpy.NewRouter(
				harpy.WithRoute("Get", server.Get),
				harpy.WithRoute("Set", harpy.NoResult(server.Set)),
			),
		},
	)
}

// KeyValueServer is a very basic key/value store with a JSON-RPC interface.
type KeyValueServer struct {
	m      sync.RWMutex
	values map[string]string
}

// GetParams contains parameters for the "Get" method.
type GetParams struct {
	Key string `json:"key"`
}

// SetParams contains parameters for the "Set" method.
type SetParams struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Get returns the value associated with a key.
//
// It returns an application-defined error if there is no value associated with
// this key.
func (s *KeyValueServer) Get(_ context.Context, params GetParams) (string, error) {
	s.m.RLock()
	defer s.m.RUnlock()

	if value, ok := s.values[params.Key]; ok {
		return value, nil
	}

	return "", harpy.NewError(
		100, // 100 is our example application's error code for "no such key"
		harpy.WithMessage("no such key"),
	)
}

// Set associates a value with a key.
func (s *KeyValueServer) Set(_ context.Context, params SetParams) error {
	s.m.Lock()
	defer s.m.Unlock()

	if s.values == nil {
		s.values = map[string]string{}
	}

	s.values[params.Key] = params.Value

	return nil
}
