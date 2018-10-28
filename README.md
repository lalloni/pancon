# pancon

Simple tool for converting between TOML, YAML and JSON.

## Use

Convert file:

```sh
pancon -i <infile> -o <outfile>
```

Used that way, `pancon` tries to guess input and output file formats from extension.

You can specify formats too:

```sh
pancon -i <infile> -d <informat> -o <outfile> -e <outformat>
```

Where `<informat>` and `<outformat>` can be one of:

- json
- yaml
- toml

When `<infile>` and/or `<outfile>` are not specified or are specified as a single "`-`",
`pancon` will use stdin or stdout for reading from and/or writing to.

## Install

From source:

```sh
go get github.com/lalloni/pancon
```
