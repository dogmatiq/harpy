package harpy

import (
	"encoding/json"
	"io"
	"strings"
)

// isJSONError returns true if err indicates a JSON parse failure of some kind.
func isJSONError(err error) bool {
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

// unmarshalJSON unmarshals JSON content from r into v. The main reason for this
// function is to disallow unknown fields.
func unmarshalJSON(r io.Reader, v interface{}) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	return dec.Decode(&v)
}
