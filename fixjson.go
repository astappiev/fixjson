// Package fixjson translates broken or non-standard JSON in src into valid JSON.
// The input is not modified; a new byte slice is returned.
package fixjson

import (
	"encoding/json"
	"fmt"
)

// ToJSON translates broken or non-standard JSON in src into valid JSON.
// The input is not modified; a new byte slice is returned.
func ToJSON(src []byte) []byte {
	return translate(src)
}

// Unmarshal tries to fix the JSON first, and if that succeeds but unmarshaling
// the fixed form fails, it falls back to the original raw input.
func Unmarshal(data []byte, v any) error {
	fixedData := ToJSON(data)

	if err := json.Unmarshal(fixedData, v); err != nil {
		if err := json.Unmarshal(data, v); err != nil {
			return describeError(err, data)
		}
	}
	return nil
}

// FallbackUnmarshal tries to unmarshal the data as valid JSON first.
// If that fails, it falls back to fixing it and unmarshaling the fixed JSON.
func FallbackUnmarshal(data []byte, v any) error {
	if err := json.Unmarshal(data, v); err != nil {
		fixedData := ToJSON(data)
		if err := json.Unmarshal(fixedData, v); err != nil {
			return describeError(err, fixedData)
		}
	}
	return nil
}

func describeError(err error, src []byte) error {
	clip := func(offset int64) string {
		if offset <= 0 {
			return ""
		}
		i := min(int(offset), len(src))
		start := i - 40
		if start < 0 {
			start = 0
		}
		return string(src[start:i])
	}

	switch t := err.(type) {
	case *json.SyntaxError:
		jsn := clip(t.Offset)
		jsn += "<--(see the invalid character)"
		return fmt.Errorf("invalid character at %v\n %s: %w", t.Offset, jsn, err)
	case *json.UnmarshalTypeError:
		jsn := clip(t.Offset)
		jsn += "<--(see the invalid type)"
		return fmt.Errorf("invalid value at %v\n %s: %w", t.Offset, jsn, err)
	default:
		return err
	}
}
