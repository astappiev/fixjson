# fixjson

[![GoDoc](https://img.shields.io/badge/api-reference-blue.svg?style=flat-square)](https://pkg.go.dev/github.com/astappiev/fixjson) 

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

## Getting Started

### Installing

```sh
$ go get -u github.com/astappiev/fixjson
```

This will retrieve the library.

### Example

There's a provided function `fixjson.ToJSON`, which does the conversion.

```go
data := `
{
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
}
`

err := json.Unmarshal(fixjson.ToJSON(data), &config)
```
