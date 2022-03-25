package harpy_test

import (
	"context"
	"fmt"

	"github.com/dogmatiq/harpy"
)

func ExampleNewRouter() {
	// Define a handler that returns the length of "positional" parameters.
	handler := func(ctx context.Context, params []string) (int, error) {
		return len(params), nil
	}

	// Create a router that routes requests for the "Len" method to the handler
	// function defined above.
	router := harpy.NewRouter(
		harpy.WithRoute("Len", handler),
	)

	fmt.Println(router.HasRoute("Len"))
	// Output: true
}

func ExampleNoResult() {
	// Define a handler that does not return a result value (just an error).
	handler := func(ctx context.Context, params []string) error {
		// perform some action
		return nil
	}

	router := harpy.NewRouter(
		// Create a route for the "PerformAction" that routes to the handler
		// function defined above.
		harpy.WithRoute("PerformAction", harpy.NoResult(handler)),
	)

	fmt.Println(router.HasRoute("PerformAction"))
	// Output: true
}
