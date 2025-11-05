package cbor

import (
	"github.com/fxamacker/cbor/v2"
)

// Encode encodes a value to CBOR.
// This handles nil maps by encoding them as empty CBOR maps.
func Encode(v any) ([]byte, error) {
	// Special handling for nil maps to ensure they encode as empty CBOR maps
	if m, ok := v.(map[string]any); ok && m == nil {
		return cbor.Marshal(map[string]any{})
	}
	return cbor.Marshal(v)
}
