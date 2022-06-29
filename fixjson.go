package fixjson

import (
	"encoding/json"
	"fmt"
	"log"
)

// ToJSON fixes the input to a valid JSON
func ToJSON(src []byte) []byte {
	return translate(src)
}

// Unmarshal tries to fix the JSON and then unmarshal it
func Unmarshal(data []byte, v any) error {
	return fixAndUnmarshal(data, v, true)
}

// FallbackUnmarshal tries to unmarshal the data as valid JSON, if that fails, it tries to fix it and unmarshal fixed
func FallbackUnmarshal(data []byte, v any) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fixAndUnmarshal(data, v, false)
	}
	return nil
}

func fixAndUnmarshal(data []byte, v any, fallback bool) error {
	fixedData := ToJSON(data)

	if err := json.Unmarshal(fixedData, v); err != nil {
		// we failed to fix the json, we can't predict everything
		if fallback {
			if err := json.Unmarshal(data, v); err != nil {
				return descError(err, data, fixedData)
			} else {
				// wow, the json was valid, but while fixing it, we broke it!
				log.Print("fixjson got valid JSON that was broken during transform.")
				log.Println("We have managed to catch that, but you may create an issues about that: https://github.com/astappiev/fixjson/issues")
				log.Println("The JSON: " + string(data))
			}
		} else {
			return descError(err, data, fixedData)
		}
	}
	return nil
}

func descError(err error, data []byte, fixed []byte) error {
	switch t := err.(type) {
	case *json.SyntaxError:
		jsn := string(data[0:t.Offset]) + " [" + string(fixed[0:t.Offset]) + "]"
		jsn += "<--(see the invalid character)"
		return fmt.Errorf("invalid character at %v\n %s", t.Offset, jsn)
	case *json.UnmarshalTypeError:
		jsn := string(data[0:t.Offset]) + " [" + string(fixed[0:t.Offset]) + "]"
		jsn += "<--(see the invalid type)"
		return fmt.Errorf("invalid value at %v\n %s", t.Offset, jsn)
	default:
		return err
	}
}
