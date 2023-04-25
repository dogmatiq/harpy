package harpy

import "github.com/dogmatiq/harpy/internal/jsonx"

type option struct {
	jsonx.UnmarshalOptionFunc
	routerOptionFunc
}
