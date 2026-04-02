# fixjson

[![Go Reference](https://pkg.go.dev/badge/github.com/astappiev/fixjson.svg)](https://pkg.go.dev/github.com/astappiev/fixjson)

**fixjson** is a Go package that converts broken, nonstandard or nonofficial format (e.g. jsonc) to standard json.
As an author of this package, I kindly ask you, stop doing that shit and use standard json whenever possible.

The features of the package are:
- remove comments, single line (`// text`) or multiline (`/* text */`)
- remove trailing commas, from arrays and objects
- add missing commas after string literals at the end of a line in arrays and objects
- remove unnecessary and invalid escape character of single quotes (`'`)
- escape quotes `"` in string literals, if they are placed not at the end of a line
- fixes html encoded quotes in json
- replace new line (`\n`) characters in string literals with `\\` following by `n`
- replace tabs (`\t`) characters with `\\` following by `t`

```text
{
  /* Example block                                  // multiline comment
   comment */
  "propertyName": {
    "name": "No, you can\'t do that in json",       // invalid escape character
    "value": 5432, // counter                       // inline comment           
    "username": "josh"                              // missing comma
    "password": "pass123",                          // trailing comma
  },

  "title": "Hello                                   // new line in string literal
    World",
}
```

## Install

```sh
go get github.com/astappiev/fixjson@latest
```

## Usage

```go
package main

import (
	"encoding/json"
	"fmt"

	"github.com/astappiev/fixjson"
)

func main() {
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
	if err := fixjson.Unmarshal(raw, &out); err != nil {
		panic(err)
	}

	fmt.Println(out["title"])

	// Or fix first, then unmarshal manually.
	fixed := fixjson.ToJSON(raw)
	_ = json.Unmarshal(fixed, &out)
}
```

## API

- `ToJSON(src []byte) []byte`: translates input into valid JSON bytes
- `Unmarshal(data []byte, v any) error`: fixes then unmarshals
- `FallbackUnmarshal(data []byte, v any) error`: tries standard unmarshal first, then fix+unmarshal
