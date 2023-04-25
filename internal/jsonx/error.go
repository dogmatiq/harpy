package jsonx

import (
	"encoding/json"
	"strings"
)

// IsParseError returns true if err indicates a JSON parse failure of some kind.
func IsParseError(err error) bool {
	switch err.(type) {
	case nil:
		return false
	case *json.SyntaxError:
		return true
	case *json.UnmarshalTypeError:
		return true
	default:
		// Unfortunately, some JSON errors do not have distinct types. For
		// example, when parsing using a decoder with DisallowUnknownFields()
		// enabled an unexpected field is reported using the equivalent of:
		//
		//   errors.New(`json: unknown field "<field name>"`)
		return strings.HasPrefix(err.Error(), "json:")
	}
}
