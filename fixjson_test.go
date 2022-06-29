package fixjson

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
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
