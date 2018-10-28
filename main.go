// MIT License
//
// Copyright (c) 2018 Pablo Lalloni
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/alecthomas/kingpin"
	"gopkg.in/yaml.v2"
)

const (
	yamlFormat = "yaml"
	jsonFormat = "json"
	tomlFormat = "toml"
)

var formats = []string{yamlFormat, jsonFormat, tomlFormat}

type Coder struct {
	Encode func(w io.Writer, v interface{}) error
	Decode func(v interface{}, r io.Reader) error
}

var coders = map[string]*Coder{
	yamlFormat: &Coder{Encode: writeYAML, Decode: readYAML},
	jsonFormat: &Coder{Encode: writeJSON, Decode: readJSON},
	tomlFormat: &Coder{Encode: writeTOML, Decode: readTOML},
}

var help = `Simple config/markup converter.

Can understand ` + strings.Join(formats, ", ") + ` formats.

Will read from stdin if input is unspecified or "-".

Will write to stdout if output is unspecified or "-".

Can be used as a pipe filter if both input and output are unspecified or "-".
`

func main() {

	app := kingpin.New(filepath.Base(os.Args[0]), help)

	inputFormat := app.Flag("decode", "Input format.").Short('d').PlaceHolder("FORMAT").Enum(formats...)
	outputFormat := app.Flag("encode", "Output format.").Short('e').PlaceHolder("FORMAT").Enum(formats...)
	inputFile := app.Flag("input", "File to read input from.").Short('i').PlaceHolder("PATH").String()
	outputFile := app.Flag("output", "File to write output to.").Short('o').PlaceHolder("PATH").String()

	kingpin.MustParse(app.Parse(os.Args[1:]))

	incoder, err := coder(*inputFile, *inputFormat, "reading from stdin")
	app.FatalIfError(err, "input")

	outcoder, err := coder(*outputFile, *outputFormat, "writing to stdout")
	app.FatalIfError(err, "output")

	infile, incloser, err := file(*inputFile, *inputFormat, os.Stdin, os.Open)
	app.FatalIfError(err, "opening input")
	defer func() { app.FatalIfError(incloser(), "closing input file") }()

	outfile, outcloser, err := file(*outputFile, *outputFormat, os.Stdout, os.Create)
	app.FatalIfError(err, "opening output")
	defer func() { app.FatalIfError(outcloser(), "closing output file") }()

	data := make(map[string]interface{})

	app.FatalIfError(incoder.Decode(&data, infile), "decoding")
	app.FatalIfError(outcoder.Encode(outfile, &data), "encoding")

}

func file(file, format string, defaultfile *os.File, opener func(string) (*os.File, error)) (*os.File, func() error, error) {
	if file == "" || file == "-" {
		return defaultfile, func() error { return nil }, nil
	}
	f, err := opener(file)
	if err != nil {
		return nil, nil, fmt.Errorf("opening %s: %s", file, err)
	}
	return f, func() error { return f.Close() }, nil

}

func coder(file, format, action string) (*Coder, error) {
	if format == "" {
		if file == "" || file == "-" {
			return nil, fmt.Errorf("format is required when %s", action)
		}
		f, err := guessformat(file)
		if err != nil {
			return nil, fmt.Errorf("guessing format: %s", err)
		}
		format = f
	}
	if c, ok := coders[format]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("unknown input format %s", format)
}

func guessformat(file string) (string, error) {
	e := strings.ToLower(filepath.Ext(file))
	if e != "" {
		e = e[1:]
		// normalize some commonly seen file "extensions"
		switch e {
		case "js":
			e = jsonFormat
		case "yml":
			e = yamlFormat
		case "tml":
			e = tomlFormat
		}
	}
	if _, ok := coders[e]; ok {
		return e, nil
	}
	return "", fmt.Errorf("can not detect format from file %s extension", file)
}

func readTOML(v interface{}, r io.Reader) error {
	_, err := toml.DecodeReader(r, &v)
	return err
}

func writeTOML(w io.Writer, v interface{}) error {
	return toml.NewEncoder(w).Encode(v)
}

func readYAML(v interface{}, r io.Reader) error {
	return yaml.NewDecoder(r).Decode(&v)
}

func writeYAML(w io.Writer, v interface{}) error {
	return yaml.NewEncoder(w).Encode(v)
}

func readJSON(v interface{}, r io.Reader) error {
	return json.NewDecoder(r).Decode(&v)
}

func writeJSON(w io.Writer, v interface{}) error {
	return json.NewEncoder(w).Encode(&v)
}
