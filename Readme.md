# Mingo - compute the minimum usable version of Go

[![Go Reference](https://pkg.go.dev/badge/github.com/bobg/mingo.svg)](https://pkg.go.dev/github.com/bobg/mingo)
[![Go Report Card](https://goreportcard.com/badge/github.com/bobg/mingo)](https://goreportcard.com/report/github.com/bobg/mingo)
[![Tests](https://github.com/bobg/mingo/actions/workflows/go.yml/badge.svg)](https://github.com/bobg/mingo/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/bobg/mingo/badge.svg?branch=main)](https://coveralls.io/github/bobg/mingo?branch=main)

This is mingo,
a library and command-line tool
for analyzing Go code
to determine the minimum version of Go necessary to compile it.

## Installation

For the command-line tool:

```sh
go install github.com/bobg/mingo/cmd/mingo@latest
```

For the library:

```sh
go get github.com/bobg/mingo@latest
```

## Usage

```sh
mingo [-v] [-deps] [-indirect] [-tests] [-check] [-api API] [DIR]
```

This command runs mingo on the Go module in the given directory DIR
(the current directory by default).
