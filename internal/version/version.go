package version

import "runtime/debug"

// Version is the current Harpy version.
var Version = "0.0.0-dev"

func init() {
	// Look through the binary's dependencies to find the current Harpy version.
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/dogmatiq/harpy" {
				Version = dep.Version
			}
		}
	}
}
