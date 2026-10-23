package fixjson

import (
	"encoding/json"
	"errors"
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

func TestDescribeErrorOffsetOutOfRangeDoesNotPanic(t *testing.T) {
	err := &json.SyntaxError{Offset: 1000}

	assertNotPanics(t, func() {
		desc := describeError(err, []byte("{\"a\":1}"))
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
				t.Parallel()
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
					t.Logf("wrote mismatch to %s", fixedPath+".new")
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

func TestDescribeErrorUnmarshalTypeError(t *testing.T) {
	err := &json.UnmarshalTypeError{Offset: 5, Value: "string", Type: reflect.TypeOf(0)}
	desc := describeError(err, []byte(`{"a":"x"}`))

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

func TestTrailingCommaBeforeComment(t *testing.T) {
	in := `{"a":1, /*x*/ }`
	out := ToJSON([]byte(in))
	if !json.Valid(out) {
		t.Errorf("expected valid JSON, got: %s", out)
	}
	in2 := `{"a":1, // x` + "\n" + `}`
	out2 := ToJSON([]byte(in2))
	if !json.Valid(out2) {
		t.Errorf("expected valid JSON, got: %s", out2)
	}
}

func TestMissingCommaAcrossComment(t *testing.T) {
	in := `{"a": "1" // x` + "\n" + `"b": "2"}`
	out := ToJSON([]byte(in))
	if !json.Valid(out) {
		t.Errorf("expected valid JSON, got: %s", out)
	}
}

func TestOddEmbeddedQuotes(t *testing.T) {
	in := `{"x": "a"b"}`
	out := ToJSON([]byte(in))
	if !json.Valid(out) {
		t.Errorf("expected valid JSON, got: %s", out)
	}
}

func TestEscapedQuoteStateLeak(t *testing.T) {
	in := `{"a": 1, "b": "x &quot; y"}`
	out := ToJSON([]byte(in))
	if !json.Valid(out) {
		t.Errorf("expected valid JSON, got: %s", out)
	}
}

func TestRepairTruncated(t *testing.T) {
	in := `{"a": "hello`
	out := ToJSON([]byte(in))
	if !json.Valid(out) {
		t.Errorf("expected valid JSON, got: %s", out)
	}
	// Also test arrays and objects
	in2 := `[{"a": "hello"`
	out2 := ToJSON([]byte(in2))
	if !json.Valid(out2) {
		t.Errorf("expected valid JSON, got: %s", out2)
	}
}

// assertFixes checks that ToJSON turns each input into the expected JSON.
func assertFixes(t *testing.T, cases map[string]string) {
	t.Helper()
	for in, expect := range cases {
		t.Run(in, func(t *testing.T) {
			assertJSONEq(t, expect, string(ToJSON([]byte(in))))
		})
	}
}

func TestReadmeExample(t *testing.T) {
	raw := []byte(`{
  /* Example block
   comment */
  "propertyName": {
    "name": "No, you can\'t do that in json",
    "value": 5432, // counter
    "username": "josh"
    "password": "pass123",
  },

  "title": "Hello
    World",
}`)

	var out map[string]any
	if err := Unmarshal(raw, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["title"] != "Hello\n    World" {
		t.Errorf("unexpected title %q", out["title"])
	}
}

func TestNewLineBeforeTrailingComma(t *testing.T) {
	cases := map[string]string{
		"{\"a\": \"x\ny\",\n}": `{"a": "x\ny"}`,
		"[\"a\nb\",]":          `["a\nb"]`,
		"[\"a\nb\",":           `["a\nb"]`,
	}
	assertFixes(t, cases)
}

func TestCommentLikeTextInString(t *testing.T) {
	cases := map[string]string{
		`{"html": "<img src="//cdn.x.com/a.png">", "b": 1}`:  `{"html": "<img src=\"//cdn.x.com/a.png\">", "b": 1}`,
		"{\"t\": \"Issue \"#42\" fixed\",\n\"b\":1}":         `{"t": "Issue \"#42\" fixed", "b": 1}`,
		"{\n \"html\": \"<a href=\"//cdn.com/x\">x</a>\"\n}": `{"html": "<a href=\"//cdn.com/x\">x</a>"}`,
		"{\n \"t\": \"fixed in \"#42\"\n}":                   `{"t": "fixed in \"#42"}`,
		"[\n \"a \"//b\" c\",\n \"d\"\n]":                    `["a \"//b\" c", "d"]`,
	}
	assertFixes(t, cases)
}

func TestCommentAfterClosingQuote(t *testing.T) {
	cases := map[string]string{
		"{\"a\":\"x\"// c\n,\"b\":1}": `{"a":"x","b":1}`,
		"{\"a\":\"x\"// c\n}":         `{"a":"x"}`,
		"[\"x\"//c\n]":                `["x"]`,
		"[\"x\"# c\n]":                `["x"]`,
		`{"a":"x"/* c */,"b":1}`:      `{"a":"x","b":1}`,
		`{"a"/*c*/:1}`:                `{"a":1}`,
		`["a"/*c*/"b"]`:               `["a","b"]`,
		`{"a":"x"// c`:                `{"a":"x"}`,
		"[\"a\"//first\n\"b\"]":       `["a","b"]`,
		"{\"a\":\"x\"#note\n\"b\":1}": `{"a":"x","b":1}`,
		`["x"//c`:                     `["x"]`,
	}
	assertFixes(t, cases)
}

func TestCommaAfterQuoteWithoutClosingQuote(t *testing.T) {
	// the quote must not be escaped, that would pull the rest of the input into the string literal
	for _, in := range []string{`{"a":"x",b:1}`, `["a",NaN]`, `{"a":["x",1x]}`, `{"a":"x",b:1,"c":2}`} {
		var out any
		if err := Unmarshal([]byte(in), &out); err == nil {
			t.Errorf("%s: expected an error, got %v", in, out)
		}
	}
}

func TestQuoteFollowedByCommaInText(t *testing.T) {
	cases := map[string]string{
		`{"a": "he said "yes", then left"}`: `{"a": "he said \"yes\", then left"}`,
		`{"a": "x "y", "b": "z"}`:           `{"a": "x \"y", "b": "z"}`,
		`{"a": "x", "b": [1, true, null]}`:  `{"a": "x", "b": [1, true, null]}`,
	}
	assertFixes(t, cases)
}

func TestEscapedControlCharacters(t *testing.T) {
	cases := map[string]string{
		"{\"a\": \"x\\\ty\", \"b\": \"x\\\ny\"}": `{"a": "x\ty", "b": "x\ny"}`,
	}
	assertFixes(t, cases)
}

func TestRepairTruncatedAtBoundaries(t *testing.T) {
	cases := map[string]string{
		`{"a":1,`:       `{"a":1}`,
		`[1,`:           `[1]`,
		`{"a":1, // x`:  `{"a":1}`,
		`{"a":"x\`:      `{"a":"x"}`,
		`{"a`:           `{"a":null}`,
		`{"a":`:         `{"a":null}`,
		`{"a" `:         `{"a":null}`,
		`{"a":{"b"`:     `{"a":{"b":null}}`,
		`{"a":"x"`:      `{"a":"x"}`,
		`["a"`:          `["a"]`,
		`{"a":"q\"x"`:   `{"a":"q\"x"}`,
		`{"a":[1,{"b":`: `{"a":[1,{"b":null}]}`,
		`{"a":"x\u00`:   `{"a":"x"}`,
		`["a\u`:         `["a"]`,
		`["a\\u00`:      `["a\\u00"]`,
		`["\u00e9`:      `["\u00e9"]`,
		`{"a":tr`:       `{"a":true}`,
		`{"a":nul `:     `{"a":null}`,
		`[1,f`:          `[1,false]`,
		`{"a":1.`:       `{"a":1}`,
		`{"a":1.5e-`:    `{"a":1.5}`,
		`{"a":-`:        `{"a":null}`,
		`[1,-`:          `[1]`,
		`{"a":"x",tr`:   `{"a":"x"}`,
		`{tr`:           `{}`,
	}
	assertFixes(t, cases)
}

func TestTypeErrorIsReported(t *testing.T) {
	cases := map[string]struct {
		unmarshal func([]byte, any) error
		in        string
	}{
		"Unmarshal":         {Unmarshal, `{"count": "2",}`},
		"FallbackUnmarshal": {FallbackUnmarshal, `{"count": "2"}`}, // fixing is skipped
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var out struct {
				Count int `json:"count"`
			}
			err := c.unmarshal([]byte(c.in), &out)
			var typeErr *json.UnmarshalTypeError
			if !errors.As(err, &typeErr) {
				t.Errorf("expected an UnmarshalTypeError, got %v", err)
			}
		})
	}
}

func FuzzToJSON(f *testing.F) {
	f.Add([]byte(`{"a":1,}`))
	f.Add([]byte(`{"a": "1" // x` + "\n" + `"b": "2"}`))
	f.Add([]byte(`{"x": "a"b"}`))
	f.Add([]byte(`{"a": "hello`))
	f.Fuzz(func(t *testing.T, src []byte) {
		_ = ToJSON(src) // must not panic
		if json.Valid(src) && !json.Valid(ToJSON(src)) {
			t.Errorf("valid input broken: %q -> %q", src, ToJSON(src))
		}
	})
}
