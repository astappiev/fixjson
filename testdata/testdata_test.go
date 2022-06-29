package fixjson

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/astappiev/fixjson"
)

var invalidSuffix = "-invalid.json"
var fixedSuffix = "-fixed.json"

func TestTestdata(t *testing.T) {
	t.Parallel()

	_ = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		assert.NoError(t, err)

		if !info.IsDir() && strings.HasSuffix(info.Name(), invalidSuffix) {
			t.Run(info.Name(), func(t *testing.T) {
				invalidJson, err := ioutil.ReadFile(path)
				assert.NoError(t, err)

				fixedPath := strings.Replace(path, invalidSuffix, fixedSuffix, 1)
				expectedJson, err := ioutil.ReadFile(fixedPath)
				assert.NoError(t, err)

				actualJson := fixjson.ToJSON(invalidJson)
				assert.True(t, json.Valid(actualJson))

				if !assert.JSONEq(t, string(expectedJson), string(actualJson)) {
					assert.NoError(t, ioutil.WriteFile(fixedPath+".new", actualJson, 0644))
				}
			})
		}
		return nil
	})
}
