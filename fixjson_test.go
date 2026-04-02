package fixjson

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmpty(t *testing.T) {
	broken := `{}`
	actual := ToJSON([]byte(broken))
	assert.Equal(t, broken, string(actual))
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

	expect := `{  
    "c": 3,"b":3, 
    
    "d\\\"\"e": [ 1,  3, 4 ]
  }`

	actual := ToJSON([]byte(broken))
	assert.True(t, json.Valid(actual))
	assert.Equal(t, expect, string(actual))
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
	assert.True(t, json.Valid(actual))
	assert.Equal(t, expect, string(actual))
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
	assert.True(t, json.Valid(actual))
	assert.Equal(t, expect, string(actual))
}

func TestInvalidEscapeChar(t *testing.T) {
	broken := `{
    "description": "This meal has it all\u2014flavorful wild rice. Don\'t have time to dedicate hours.",
  }`
	expect := `{
    "description": "This meal has it all\u2014flavorful wild rice. Don't have time to dedicate hours."
  }`

	actual := ToJSON([]byte(broken))
	assert.True(t, json.Valid(actual))
	assert.Equal(t, expect, string(actual))
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
	assert.True(t, json.Valid(actual))
	assert.Equal(t, expect, string(actual))
}

func TestEscaped(t *testing.T) {
	broken := `{
    &quot;description&quot;: &quot;This meal has it all\u2014flavorful wild rice. Don\'t have time to dedicate hours.&quot;,
  }`
	expect := `{
    "description": "This meal has it all\u2014flavorful wild rice. Don't have time to dedicate hours."
  }`

	actual := ToJSON([]byte(broken))
	assert.True(t, json.Valid(actual))
	assert.Equal(t, expect, string(actual))
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
	expect := `{
    "recipeCuisine": "European\n\t\tcuisine",

    
    "recipeIngredient": ["200 g Beets (baked or boiled)\n      ",
 "150 g Feta\n\t  "
]
  }`

	actual := ToJSON([]byte(broken))
	assert.True(t, json.Valid(actual))
	assert.Equal(t, expect, string(actual))
}

func TestToJSONTrailingSlashDoesNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		_ = ToJSON([]byte("{\"a\":1}/"))
	})
}

func TestDescErrorOffsetOutOfRangeDoesNotPanic(t *testing.T) {
	err := &json.SyntaxError{Offset: 1000}

	assert.NotPanics(t, func() {
		desc := descError(err, []byte("{\"a\":1}"), []byte("{"))
		assert.Contains(t, desc.Error(), "invalid character at 1000")
	})
}

const invalidSuffix = "-invalid.json"
const fixedSuffix = "-fixed.json"

func TestTestdata(t *testing.T) {
	t.Parallel()

	err := filepath.WalkDir("testdata", func(path string, d fs.DirEntry, err error) error {
		require.NoError(t, err)

		if !d.IsDir() && strings.HasSuffix(d.Name(), invalidSuffix) {
			t.Run(d.Name(), func(t *testing.T) {
				invalidJson, err := os.ReadFile(path)
				require.NoError(t, err)

				fixedPath := strings.Replace(path, invalidSuffix, fixedSuffix, 1)
				expectedJson, err := os.ReadFile(fixedPath)
				require.NoError(t, err)

				actualJson := ToJSON(invalidJson)
				assert.True(t, json.Valid(actualJson))

				if !assert.JSONEq(t, string(expectedJson), string(actualJson)) {
					assert.NoError(t, os.WriteFile(fixedPath+".new", actualJson, 0o644))
				}
			})
		}
		return nil
	})
	assert.NoError(t, err)
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
	assert.NoError(t, err)
	assert.Equal(t, "Cake", out.Name)
	assert.Equal(t, 2, out.Count)
}

func TestFallbackUnmarshalValidJSON(t *testing.T) {
	valid := []byte(`{"ok":true,"n":1}`)

	var out map[string]any
	err := FallbackUnmarshal(valid, &out)
	assert.NoError(t, err)
	assert.Equal(t, true, out["ok"])
	assert.Equal(t, float64(1), out["n"])
}

func TestFallbackUnmarshalReturnsDescriptiveError(t *testing.T) {
	invalid := []byte(`{"a": }`)

	var out map[string]any
	err := FallbackUnmarshal(invalid, &out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character at")
	assert.Contains(t, err.Error(), "<--(see the invalid character)")
}

func TestDescErrorUnmarshalTypeError(t *testing.T) {
	err := &json.UnmarshalTypeError{Offset: 5, Value: "string", Type: reflect.TypeOf(0)}
	desc := descError(err, []byte(`{"a":"x"}`), []byte(`{"a":"x"}`))

	assert.Error(t, desc)
	assert.Contains(t, desc.Error(), "invalid value at 5")
	assert.Contains(t, desc.Error(), "<--(see the invalid type)")
}

func TestHashComments(t *testing.T) {
	broken := "{\n# heading comment\n\"a\": 1,\n}"
	expect := "{\n\n\"a\": 1\n}"

	actual := ToJSON([]byte(broken))
	assert.True(t, json.Valid(actual))
	assert.Equal(t, expect, string(actual))
}
