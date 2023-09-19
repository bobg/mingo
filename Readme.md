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

For library usage please see
[the package doc](https://pkg.go.dev/github.com/bobg/mingo).

Command-line usage:

```sh
mingo [-v] [-deps] [-indirect] [-tests] [-check] [-api API] [DIR]
```

This command runs mingo on the Go module in the given directory DIR
(the current directory by default).

The flags and their meanings are:

----------------------------------------------------------------------------------------------
| -v         | Run verbosely                                                                 |
| -deps      | Include dependencies                                                          |
| -indirect  | With -deps, include indirect dependencies                                     |
| -tests     | Include tests                                                                 |
| -check     | Check that the module declares the right version of Go                        |
| -api API   | Find the Go API files in the directory API instead of the default $GOROOT/api |
----------------------------------------------------------------------------------------------

Normal output is the lowest minor version of Go
(the x in Go 1.x)
that is safe to declare in the `go` directive of the module’s `go.mod` file.

Running with `-check` causes mingo to exit with a 0 status code and no output
if the module declares the correct version of Go,
or a non-zero status and an error message otherwise.

Including dependencies with `-deps`
allows `go` directives in imported modules’ `go.mod` files
to change the result.
Normally this includes only direct imports,
but with the addition of `-indirect` it includes indirect imports too.

## Discussion

What version of Go should you declare in your `go.mod` file?

For maximum compatibility it should be the oldest version of Go that can compile your code.

For example, if your code uses a `for range` statement that does not include a variable assignment,
you need at least Go 1.4,
which [first introduced](https://go.dev/doc/go1.4#language) variable-free `for range` statements.
And if you use the `context` package from the standard library,
you need at least [Go 1.7](https://go.dev/doc/go1.7#context).
On the other hand if you use the function `context.Cause`
that requirement bumps up to [Go 1.20](https://go.dev/doc/go1.20#minor_library_changes).

One thing you should _not_ do is to routinely increase the version in your `go.mod`
when a new version of Go comes out.
When you do you risk breaking compatibility for some of your callers.

Practically speaking there’s no point declaring a version of Go earlier than 1.11 in your `go.mod`,
since [that’s the first version](https://go.dev/doc/go1.11#modules) that understood `go.mod` files.
But mingo will still report earlier version numbers when warranted
(somewhat pedantically).
