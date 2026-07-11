package fixjson

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// assertJSONEq checks that two JSON strings are semantically equal.
// Returns true if equal, false otherwise (and logs an error).
func assertJSONEq(t *testing.T, expected, actual string) bool {
	t.Helper()
	var e, a any
	if err := json.Unmarshal([]byte(expected), &e); err != nil {
		t.Errorf("failed to parse expected JSON: %v", err)
		return false
	}
	if err := json.Unmarshal([]byte(actual), &a); err != nil {
		t.Errorf("failed to parse actual JSON: %v", err)
		return false
	}
	if !reflect.DeepEqual(e, a) {
		t.Errorf("JSON not equal\nexpected: %s\nactual:   %s", expected, actual)
		return false
	}
	return true
}

// assertNotPanics verifies that fn does not panic.
func assertNotPanics(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("unexpected panic: %v", r)
		}
	}()
	fn()
}

func TestEmpty(t *testing.T) {
	broken := `{}`
	actual := ToJSON([]byte(broken))
	if broken != string(actual) {
		t.Errorf("expected %q, got %q", broken, string(actual))
	}
}

func TestComments(t *testing.T) {
	broken := `{  //	hello
    "c": 3,"b":3, // jello
    /* SOME
       LIKE
       IT
       HAUT */
    "d\\\"\"e": [ 1, /* 2 */ 3, 4, ],
  }`

	expect := "{  \n    \"c\": 3,\"b\":3, \n    \n    \"d\\\\\\\"\\\"e\": [ 1,  3, 4 ]\n  }"

	actual := ToJSON([]byte(broken))
	if !json.Valid(actual) {
		t.Error("result is not valid JSON")
	}
	if expect != string(actual) {
		t.Errorf("expected %q, got %q", expect, string(actual))
	}
}

func TestTrailingCommas(t *testing.T) {
	broken := `{
	"id": "0001",
	"type": "donut",
	"name": "Cake",
	}`
	expect := `{
	"id": "0001",
	"type": "donut",
	"name": "Cake"
	}`

	actual := ToJSON([]byte(broken))
	if !json.Valid(actual) {
		t.Error("result is not valid JSON")
	}
	if expect != string(actual) {
		t.Errorf("expected %q, got %q", expect, string(actual))
	}
}

func TestMissingCommas(t *testing.T) {
	broken := `{
	"id": "0001"
	"type": "donut"
	"name": "Cake"
	}`
	expect := `{
	"id": "0001",
	"type": "donut",
	"name": "Cake"
	}`

	actual := ToJSON([]byte(broken))
	if !json.Valid(actual) {
		t.Error("result is not valid JSON")
	}
	if expect != string(actual) {
		t.Errorf("expected %q, got %q", expect, string(actual))
	}
}

func TestInvalidEscapeChar(t *testing.T) {
	broken := `{
    "description": "This meal has it all\u2014flavorful wild rice. Don\'t have time to dedicate hours.",
  }`
	expect := `{
    "description": "This meal has it all\u2014flavorful wild rice. Don't have time to dedicate hours."
  }`

	actual := ToJSON([]byte(broken))
	if !json.Valid(actual) {
		t.Error("result is not valid JSON")
	}
	if expect != string(actual) {
		t.Errorf("expected %q, got %q", expect, string(actual))
	}
}

func TestMissingEscaping(t *testing.T) {
	broken := `{
	"recipeInstructions": [
	  {
	  "text": "Именно такой режим я использовал в своей <a rel="nofollow" href="https://www.panasonic.com/ua/consumer/kitchen-appliances/steam-ovens/nu-sc300bzpe.html" target="_blank">печи Panasonic</a>.",
	  },]
}`
	expect := `{
	"recipeInstructions": [
	  {
	  "text": "Именно такой режим я использовал в своей <a rel=\"nofollow\" href=\"https://www.panasonic.com/ua/consumer/kitchen-appliances/steam-ovens/nu-sc300bzpe.html\" target=\"_blank\">печи Panasonic</a>."
	  }]
}`

	actual := ToJSON([]byte(broken))
	if !json.Valid(actual) {
		t.Error("result is not valid JSON")
	}
	if expect != string(actual) {
		t.Errorf("expected %q, got %q", expect, string(actual))
	}
}

func TestEscaped(t *testing.T) {
	broken := `{
    &quot;description&quot;: &quot;This meal has it all\u2014flavorful wild rice. Don\'t have time to dedicate hours.&quot;,
  }`
	expect := `{
    "description": "This meal has it all\u2014flavorful wild rice. Don't have time to dedicate hours."
  }`

	actual := ToJSON([]byte(broken))
	if !json.Valid(actual) {
		t.Error("result is not valid JSON")
	}
	if expect != string(actual) {
		t.Errorf("expected %q, got %q", expect, string(actual))
	}
}

func TestNewLines(t *testing.T) {
	broken := `{
    "recipeCuisine": "European
		cuisine",
    /*"nutrition": {
      "calories": "270 calories"
    },*/
    "recipeIngredient": ["200 g Beets (baked or boiled)
      ", "150 g Feta
	  "],
  }`
	expect := "{\n    \"recipeCuisine\": \"European\\n\\t\\tcuisine\",\n\n    \n    \"recipeIngredient\": [\"200 g Beets (baked or boiled)\\n      \",\n \"150 g Feta\\n\\t  \"\n]\n  }"

	actual := ToJSON([]byte(broken))
	if !json.Valid(actual) {
		t.Error("result is not valid JSON")
	}
	if expect != string(actual) {
		t.Errorf("expected %q, got %q", expect, string(actual))
	}
}

func TestToJSONTrailingSlashDoesNotPanic(t *testing.T) {
	assertNotPanics(t, func() {
		_ = ToJSON([]byte("{\"a\":1}/"))
	})
}

func TestDescErrorOffsetOutOfRangeDoesNotPanic(t *testing.T) {
	err := &json.SyntaxError{Offset: 1000}

	assertNotPanics(t, func() {
		desc := descError(err, []byte("{\"a\":1}"), []byte("{"))
		if !strings.Contains(desc.Error(), "invalid character at 1000") {
			t.Errorf("expected error to contain %q, got %q", "invalid character at 1000", desc.Error())
		}
	})
}

const invalidSuffix = "-invalid.json"
const fixedSuffix = "-fixed.json"

func TestTestdata(t *testing.T) {
	t.Parallel()

	err := filepath.WalkDir("testdata", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			t.Fatal(err)
		}

		if !d.IsDir() && strings.HasSuffix(d.Name(), invalidSuffix) {
			t.Run(d.Name(), func(t *testing.T) {
				invalidJson, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}

				fixedPath := strings.Replace(path, invalidSuffix, fixedSuffix, 1)
				expectedJson, err := os.ReadFile(fixedPath)
				if err != nil {
					t.Fatal(err)
				}

				actualJson := ToJSON(invalidJson)
				if !json.Valid(actualJson) {
					t.Error("result is not valid JSON")
				}

				if !assertJSONEq(t, string(expectedJson), string(actualJson)) {
					_ = os.WriteFile(fixedPath+".new", actualJson, 0o644)
				}
			})
		}
		return nil
	})
	if err != nil {
		t.Errorf("WalkDir error: %v", err)
	}
}

func TestUnmarshalFixesBrokenJSON(t *testing.T) {
	broken := []byte(`{
		"name": "Cake"
		"count": 2,
	}`)

	var out struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	err := Unmarshal(broken, &out)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if out.Name != "Cake" {
		t.Errorf("expected Name %q, got %q", "Cake", out.Name)
	}
	if out.Count != 2 {
		t.Errorf("expected Count %d, got %d", 2, out.Count)
	}
}

func TestFallbackUnmarshalValidJSON(t *testing.T) {
	valid := []byte(`{"ok":true,"n":1}`)

	var out map[string]any
	err := FallbackUnmarshal(valid, &out)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if out["ok"] != true {
		t.Errorf("expected out[\"ok\"] = true, got %v", out["ok"])
	}
	if out["n"] != float64(1) {
		t.Errorf("expected out[\"n\"] = 1.0, got %v", out["n"])
	}
}

func TestFallbackUnmarshalReturnsDescriptiveError(t *testing.T) {
	invalid := []byte(`{"a": }`)

	var out map[string]any
	err := FallbackUnmarshal(invalid, &out)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid character at") {
		t.Errorf("expected error to contain %q, got %q", "invalid character at", err.Error())
	}
	if !strings.Contains(err.Error(), "<--(see the invalid character)") {
		t.Errorf("expected error to contain %q, got %q", "<--(see the invalid character)", err.Error())
	}
}

func TestDescErrorUnmarshalTypeError(t *testing.T) {
	err := &json.UnmarshalTypeError{Offset: 5, Value: "string", Type: reflect.TypeOf(0)}
	desc := descError(err, []byte(`{"a":"x"}`), []byte(`{"a":"x"}`))

	if desc == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(desc.Error(), "invalid value at 5") {
		t.Errorf("expected error to contain %q, got %q", "invalid value at 5", desc.Error())
	}
	if !strings.Contains(desc.Error(), "<--(see the invalid type)") {
		t.Errorf("expected error to contain %q, got %q", "<--(see the invalid type)", desc.Error())
	}
}

func TestHashComments(t *testing.T) {
	broken := "{\n# heading comment\n\"a\": 1,\n}"
	expect := "{\n\n\"a\": 1\n}"

	actual := ToJSON([]byte(broken))
	if !json.Valid(actual) {
		t.Error("result is not valid JSON")
	}
	if expect != string(actual) {
		t.Errorf("expected %q, got %q", expect, string(actual))
	}
}
