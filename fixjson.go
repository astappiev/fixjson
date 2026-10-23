// Package fixjson translates broken or non-standard JSON in src into valid JSON.
// The input is not modified; a new byte slice is returned.
package fixjson

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ToJSON translates broken or non-standard JSON in src into valid JSON.
// The input is not modified; a new byte slice is returned.
func ToJSON(src []byte) []byte {
	return translate(src)
}

// Unmarshal fixes the JSON first and unmarshals the fixed form.
// If the fixed form is not valid JSON but the original raw input is, it unmarshals the raw input instead.
// A returned error describes the form that was unmarshaled.
func Unmarshal(data []byte, v any) error {
	fixedData := ToJSON(data)

	if err := json.Unmarshal(fixedData, v); err != nil {
		if isSyntaxError(err) && json.Valid(data) { // v is untouched, syntax is checked before decoding
			if err := json.Unmarshal(data, v); err != nil {
				return describeError(err, data)
			}
			return nil
		}
		return describeError(err, fixedData)
	}
	return nil
}

// FallbackUnmarshal tries to unmarshal the data as valid JSON first.
// If the data is not valid JSON, it falls back to fixing it and unmarshaling the fixed JSON.
func FallbackUnmarshal(data []byte, v any) error {
	err := json.Unmarshal(data, v)
	if err == nil {
		return nil
	}
	if !isSyntaxError(err) { // fixing cannot help, e.g. with a type mismatch
		return describeError(err, data)
	}

	fixedData := ToJSON(data)
	if err := json.Unmarshal(fixedData, v); err != nil {
		return describeError(err, fixedData)
	}
	return nil
}

func isSyntaxError(err error) bool {
	var syntaxErr *json.SyntaxError
	return errors.As(err, &syntaxErr)
}

func describeError(err error, src []byte) error {
	clip := func(offset int64) string {
		if offset <= 0 {
			return ""
		}
		i := min(int(offset), len(src))
		return string(src[max(i-40, 0):i])
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
