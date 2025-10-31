package cbor

import (
	"reflect"

	"github.com/fxamacker/cbor/v2"
)

var cborDecoder cbor.DecMode

func init() {
	// Configure CBOR decoder with default options
	decOpts := cbor.DecOptions{
		// When unmarshalling into an empty interface, decode maps into map[string]any (instead of map[any]any).
		DefaultMapType: reflect.TypeOf(map[string]any(nil)),

		// When unmarshalling integers into an empty interface, use signed int64. Values that overflow will report an error.
		IntDec: cbor.IntDecConvertSigned,
	}
	cborDecoder, _ = decOpts.DecMode()
}

// Decode decodes CBOR data into a value
func Decode(data []byte, v any) error {
	return cborDecoder.Unmarshal(data, v)
}

// DecodeToMap decodes CBOR data into a map with type normalization
func DecodeToMap(data []byte) (map[string]any, error) {
	var result map[string]any
	err := cborDecoder.Unmarshal(data, &result)
	return result, err
}
