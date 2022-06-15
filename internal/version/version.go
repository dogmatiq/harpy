package version

import "runtime/debug"

// Version is the current Harpy version.
var Version = "0.0.0-dev"

func init() {
	info, ok := debug.ReadBuildInfo()
	if ok {
		return
	}

	for _, dep := range info.Deps {
		if dep.Path == "github.com/dogmatiq/harpy" {
			Version = dep.Version
		}
	}
}
