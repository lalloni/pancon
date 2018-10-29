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
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/alecthomas/kingpin"
	"gopkg.in/yaml.v2"
)

var help = `Simple config/markup converter.

Will read from stdin if input is unspecified or "-".

Will write to stdout if output is unspecified or "-".

Can be used as a pipe filter if both input and output are unspecified or "-".`

type Coder struct {
	Format string
	Decode func(v interface{}, r io.Reader) error
	Encode func(w io.Writer, v interface{}) error
}

var coders = []*Coder{
	{"yaml", readYAML, writeYAML},
	{"json", readJSON, writeJSON},
	{"toml", readTOML, writeTOML},
}

var formats []string

func init() {
	decoders := []string{}
	encoders := []string{}
	for _, coder := range coders {
		formats = append(formats, coder.Format)
		if coder.Encode != nil {
			encoders = append(encoders, coder.Format)
		}
		if coder.Decode != nil {
			decoders = append(decoders, coder.Format)
		}
	}
	sort.Strings(formats)
	sort.Strings(encoders)
	sort.Strings(decoders)
	help = help + "\n\nSupported formats for input decoding: " + strings.Join(decoders, " ")
	help = help + "\n\nSupported formats for output encoding: " + strings.Join(encoders, " ")
}

func main() {

	app := kingpin.New(filepath.Base(os.Args[0]), help)

	inputFormat := app.Flag("decode", "Input format.").Short('d').PlaceHolder("FORMAT").Enum(formats...)
	outputFormat := app.Flag("encode", "Output format.").Short('e').PlaceHolder("FORMAT").Enum(formats...)
	inputFile := app.Flag("input", "File to read input from.").Short('i').PlaceHolder("PATH").String()
	outputFile := app.Flag("output", "File to write output to.").Short('o').PlaceHolder("PATH").String()

	kingpin.MustParse(app.Parse(os.Args[1:]))

	incoder, err := coder(*inputFile, *inputFormat, "reading from stdin")
	app.FatalIfError(err, "input")
	if incoder.Decode == nil {
		app.Fatalf("input format %s not supported for decoding", incoder.Format)
	}

	outcoder, err := coder(*outputFile, *outputFormat, "writing to stdout")
	app.FatalIfError(err, "output")
	if outcoder.Decode == nil {
		app.Fatalf("output format %s not supported for encoding", outcoder.Format)
	}

	infile, incloser, err := file(*inputFile, *inputFormat, os.Stdin, os.Open)
	app.FatalIfError(err, "opening input")
	defer func() { app.FatalIfError(incloser(), "closing input file") }()

	outfile, outcloser, err := file(*outputFile, *outputFormat, os.Stdout, os.Create)
	app.FatalIfError(err, "opening output")
	defer func() { app.FatalIfError(outcloser(), "closing output file") }()

	data := map[string]interface{}{}

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
	if c := coderFor(format); c != nil {
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
			e = "json"
		case "yml":
			e = "yaml"
		case "tml":
			e = "toml"
		}
	}
	if coderFor(e) != nil {
		return e, nil
	}
	return "", fmt.Errorf("can not detect format from file %s extension", file)
}

func coderFor(format string) *Coder {
	for _, coder := range coders {
		if coder.Format == format {
			return coder
		}
	}
	return nil
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
