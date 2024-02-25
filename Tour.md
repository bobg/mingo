# Let’s understand static analysis of Go code: A guided tour of “mingo”

“Static analysis” is what a program does when it parses some other program’s source code
in order to transform, report on, or operate on it in some way.
A simple example is turning the include/import/require statements in a collection of files into a dependency graph.
A more complex example is a linter,
which looks for style and correctness problems.
Your IDE does static analysis when it helps you jump to the definition of some identifier.
And of course the compiler statically analyzes source code in order to turn it into an executable binary.

The Go programming language was designed in part to be easy to parse and analyze,
for the sake of fast compilation,
and to simplify the creation of all manner of development tooling.
To that end, the Go standard library exposes some relevant packages,
such as [go/ast](https://pkg.go.dev/go/ast)
(for representing a program’s abstract syntax tree)
and [go/types](https://pkg.go.dev/go/types)
(for supplementing such a tree with type information).
There are additional static-analysis tools in the [golang.org/x/tools](https://pkg.go.dev/golang.org/x/tools) module —
about which, more below.

“Mingo” is a Go static-analysis tool that I created recently in order to answer the question,
“What is the oldest version of Go that can compile my code?”
The answer to that question is what belongs in [the go directive](https://go.dev/ref/mod#go-mod-file-go) in a Go program’s go.mod file,
but it has historically been challenging to know what to put there.

At a high level,
mingo works by parsing a Go program,
then walking its syntax tree looking for two things:

1. Language constructs introduced at a specific version of Go.
   For example, if a program contains a `for … range` statement with no variable assignment,
   it needs at least Go 1.4.
   If a bit-shift expression uses an signed integer on the right-hand side,
   it needs at least Go 1.13.
2. Identifiers added to the Go standard library at specific versions.
   For example, if the program refers to `bufio.ErrFinalToken`,
   it needs at least Go 1.6.
   If it calls `bytes.Clone`,
   it needs at least Go 1.20.

Mingo refers to this process as a “scan,”
and the behavior of the scan is controlled by a `Scanner`,
defined [here](https://github.com/bobg/mingo/blob/562b72282874015100556d6cecff601d9c9fd07a/scan.go#L18-L31).
The caller configures it with the desired settings
and then calls [its ScanDir method](https://github.com/bobg/mingo/blob/562b72282874015100556d6cecff601d9c9fd07a/scan.go#L47).
The result of that is a [Result](https://github.com/bobg/mingo/blob/562b72282874015100556d6cecff601d9c9fd07a/result.go#L11-L14),
which can report the lowest minor version of Go needed for the scanned code
(e.g., the 18 in “Go 1.18”)
and a string explaining what feature of that version made it necessary
(e.g., “generic function type”).

`Scanner.ScanDir` uses [packages.Load](https://pkg.go.dev/golang.org/x/tools/go/packages#Load)
to turn a tree of Go files into a sequence of [packages.Package](https://pkg.go.dev/golang.org/x/tools/go/packages#Package) objects.
Each encapsulates a lot of information about each Go package encountered,
including syntax trees and types.

This list of packages is passed to [Scanner.ScanPackages](https://github.com/bobg/mingo/blob/562b72282874015100556d6cecff601d9c9fd07a/scan.go#L69),
which loops over them and calls [Scanner.scanPackage](https://github.com/bobg/mingo/blob/562b72282874015100556d6cecff601d9c9fd07a/scan.go#L131) on each one
(stopping early [if it ever discovers](https://github.com/bobg/mingo/blob/562b72282874015100556d6cecff601d9c9fd07a/scan.go#L99-L101) that the maximum known version of Go is needed).

This ends up [creating a pkgScanner object](https://github.com/bobg/mingo/blob/562b72282874015100556d6cecff601d9c9fd07a/scan.go#L136-L141)
containing the `Scanner` itself and information from the `packages.Package` object.
That is then used to continue the scan
by calling [pkgScanner.file](https://github.com/bobg/mingo/blob/562b72282874015100556d6cecff601d9c9fd07a/package.go#L18)
for each file in the package
(again [stopping early](https://github.com/bobg/mingo/blob/562b72282874015100556d6cecff601d9c9fd07a/scan.go#L148-L150) if the max-known-Go-version condition is reached).

Here’s where we can delve a bit into the types from the `go/ast` package.
The syntax trees of the package are a sequence of [ast.File](https://pkg.go.dev/go/ast#File) objects,
each of which contains a `Decls` field, a list of top-level declarations
(among a few other pieces of information).
For mingo’s purposes,
`pkgScanner.file` needs only to iterate over those declarations and scan those.

The type [ast.Decl](https://pkg.go.dev/go/ast#Decl) is an interface,
implemented by these concrete types:

- [BadDecl](https://pkg.go.dev/go/ast#BadDecl)
- [GenDecl](https://pkg.go.dev/go/ast#GenDecl)
- [FuncDecl](https://pkg.go.dev/go/ast#FuncDecl)

(How do I know it’s those and no others?
`Decl` uses the Go trick of requiring implementations to define an unexported no-op method −
here it’s [declNode](https://cs.opensource.google/go/go/+/master:src/go/ast/ast.go;l=52;drc=28f1bf61b7383bd4079d77090e67b3198b75be12) −
and [these three types](https://cs.opensource.google/go/go/+/master:src/go/ast/ast.go;l=1011-1015;drc=28f1bf61b7383bd4079d77090e67b3198b75be12) are the only ones in `$GOROOT/src/go/ast` that define it.)

We don’t care about `BadDecl`,
which is a placeholder for declarations that couldn’t be parsed properly.
If there is any unparseable code,
Mingo will already have returned an error ([here](https://github.com/bobg/mingo/blob/562b72282874015100556d6cecff601d9c9fd07a/scan.go#L76-L85)).
So [pkgScanner.decl](https://github.com/bobg/mingo/blob/562b72282874015100556d6cecff601d9c9fd07a/decl.go#L9) does a type switch,
calling [pkgScanner.funcDecl](https://github.com/bobg/mingo/blob/562b72282874015100556d6cecff601d9c9fd07a/decl.go#L19) for function declarations
and [pkgScanner.genDecl](https://github.com/bobg/mingo/blob/562b72282874015100556d6cecff601d9c9fd07a/decl.go#L77)
for generalized declarations (variables, constants, etc).

So far, this tree walk has taken us from containers to the things they contain:
from a directory to a set of packages;
from a package to a set of files;
from a file to a set of declarations.
Now at last we’re getting to where some actual computation happens.

A [function declaration](https://pkg.go.dev/go/ast#FuncDecl) consists of an optional doc comment,
an optional “receiver” (if the function is a method),
a name,
a “signature” (the names and types of any parameters and results),
and a body.
The receiver, the parameters, the results, and the body we descend as before
(with calls to `pkgScanner.fieldList` and `pkgScanner.funcBody`),
but now we also get to do our first check for a version-specific feature:
generic type parameters.
If we see any,
[we know this code requires Go 1.18](https://github.com/bobg/mingo/blob/562b72282874015100556d6cecff601d9c9fd07a/decl.go#L27-L37) or later.
The call to `p.greater` updates the `Result` in the `Scanner`
(if 18 is greater than what’s already there)
and returns the result of an `isMax` check
that might allow us to quit early.

(How do we know this change requires Go 1.18?
By reading the “Changes to the language” section of each Go version’s release notes.
[Here is the one for Go 1.18](https://tip.golang.org/doc/go1.18#language).)

Let’s drill down into `pkgScanner.funcBody`,
which [checks the list of statements](https://github.com/bobg/mingo/blob/562b72282874015100556d6cecff601d9c9fd07a/expr.go#L163-L170)
that constitute a function body,
and then checks that there’s a final `return` statement.
(If there isn’t, then Go 1.1 or later is required.)

The type [ast.Stmt](https://pkg.go.dev/go/ast#Stmt),
together with `ast.Decl` and `ast.Expr`
(which we’ll see in a moment),
form the core of the `ast` package.
It’s an interface, and it’s implemented by
[a wide assortment](https://cs.opensource.google/go/go/+/master:src/go/ast/ast.go;l=849-871;drc=ef84d62cfc358ff62c60da9ceec754e7a389b5d5)
of concrete types,
representing variable assignments,
`for` loops,
conditionals,
and every other kind of statement in the Go language.
Mingo inspects each statement with
[pkgScanner.stmt](https://github.com/bobg/mingo/blob/562b72282874015100556d6cecff601d9c9fd07a/stmt.go#L10),
which is just a big type switch for dispatching to functions for each concrete statement type.
Let’s zoom in to one of those concrete types,
[ast.AssignStmt](https://pkg.go.dev/go/ast#AssignStmt),
handled by [pkgScanner.assignStmt](https://github.com/bobg/mingo/blob/562b72282874015100556d6cecff601d9c9fd07a/stmt.go#L87).

An assignment statement is a sequence of left-hand expressions
(the variables, struct fields, or other lvalues being assigned to),
a sequence of right-hand expressions
(the values to assign to them),
and an assignment operator.
This is one of the [token.Token](https://pkg.go.dev/go/token#Token) constants
from the `go/token` package,
representing operators like `=`, `:=`, `+=`, and so on.

Scanning an assignment statement involves descending into the expressions it contains, of course,
but there is [some additional logic](https://github.com/bobg/mingo/blob/562b72282874015100556d6cecff601d9c9fd07a/stmt.go#L89-L97)
for checking whether this is a bit-shift assignment
(operator `<<=` or `>>=`) and,
if it is,
whether its right-hand side is a signed rather than an unsigned integer.
In that case,
[Go 1.13 is required](https://tip.golang.org/doc/go1.13#language).

How is that check done?
By looking up the `ast.Expr` on the right-hand side of the assignment
in the `Types` map of the [types.Info](https://pkg.go.dev/go/types#Info)
contained in the `pkgScanner`.
The result of the lookup is a [types.TypeAndValue](https://pkg.go.dev/go/types#TypeAndValue).
This contains at least the type,
and for constant expressions also the value,
of expressions in the abstract syntax tree.

The `Type` in a `TypeAndValue` is [another interface](https://pkg.go.dev/go/types#Type),
representing categories of Go type.
One implementation of `Type` is [types.Array](https://pkg.go.dev/go/types#Array).
Another is [types.Map](https://pkg.go.dev/go/types#Map).
And so on.
The one we’re interested in for now
is [types.Basic](https://pkg.go.dev/go/types#Basic),
which represents booleans, numbers, and strings.
To test whether that right-hand-side expression of the bit-shift-assignment statement is signed,
[we check](https://github.com/bobg/mingo/blob/562b72282874015100556d6cecff601d9c9fd07a/package.go#L46)
whether it’s a `*types.Basic`
and then whether its `types.Integer` flag is set and its `types.IsUnsigned` flag is unset.

Let’s now take a closer look at the last of our major interface types,
[ast.Expr](https://pkg.go.dev/go/ast#Expr),
which represents every kind of expression in Go:
identifiers, pointer dereferences, multiplications,
and also types that are spelled out in the code.
This interface is implemented by [this collection](https://cs.opensource.google/go/go/+/master:src/go/ast/ast.go;l=548-573;drc=ef84d62cfc358ff62c60da9ceec754e7a389b5d5) of concrete types.

When `pkgScanner.assignStmt` needs to descend into the left-hand and right-hand expressions,
it does so by calling [pkgScanner.expr](https://github.com/bobg/mingo/blob/562b72282874015100556d6cecff601d9c9fd07a/expr.go#L11) on each of them.
This leads to another big type switch that dispatches to another set of type-specific `pkgScanner` methods.
By now the techniques for walking these data types should be familiar,
but there’s still one important thing we haven’t seen,
so let’s focus on the expression type [ast.Ident](https://pkg.go.dev/go/ast#Ident),
handled in [pkgScanner.ident](https://github.com/bobg/mingo/blob/e25314c0cc521e743eb39543db37296d4239df46/expr.go#L70).

First the function checks to see whether the identifier is `any`,
and is being used as a type,
and the type is an empty interface.
If so, the code requires Go 1.18.

Otherwise it’s time to consult the `Uses` map of the [types.Info](https://pkg.go.dev/go/types#Info)
contained in the `pkgScanner`.
This maps the identifier to the thing it’s being used to denote.
That thing −
a type, a function, a constant, etc. −
is represented by a [types.Object](https://pkg.go.dev/go/types#Object),
and from it we can get the path of the package it lives in.

Now we can use the package’s path and the identifier to do [an API history lookup](https://github.com/bobg/mingo/blob/e25314c0cc521e743eb39543db37296d4239df46/expr.go#L97).
This is where we check to see whether the code is referring to something from the standard library
that was introduced at some particular version of Go.
But how?

One thing we glossed over,
back at the beginning of `Scanner.ScanPackages`,
was [this call](https://github.com/bobg/mingo/blob/e25314c0cc521e743eb39543db37296d4239df46/scan.go#L70) to `Scanner.ensureHistory`,
which ensures that the scanner’s API history information
([this field](https://github.com/bobg/mingo/blob/e25314c0cc521e743eb39543db37296d4239df46/scan.go#L29))
is populated.
That happens [here](https://github.com/bobg/mingo/blob/e25314c0cc521e743eb39543db37296d4239df46/hist.go#L66).
It parses the files in [the api directory](https://cs.opensource.google/go/go/+/master:api/) that ships with Go
([a snapshot of which](https://github.com/bobg/mingo/tree/main/api) is [embedded](https://github.com/bobg/mingo/blob/e25314c0cc521e743eb39543db37296d4239df46/hist.go#L56-L57) into mingo itself).
Each file describes the new identifiers added to the standard library at a particular version of Go.
[Here](https://github.com/bobg/mingo/blob/e25314c0cc521e743eb39543db37296d4239df46/api/go1.7.txt#L6-L19), for example,
is the addition of the `context` package in Go 1.7.

Armed with that history,
looking up an identifier in a particular package path is relatively simple,
and happens [here](https://github.com/bobg/mingo/blob/e25314c0cc521e743eb39543db37296d4239df46/hist.go#L26).

Back in `pkgScanner.ident`,
if we get a hit from this history lookup,
we [update the required Go version](https://github.com/bobg/mingo/blob/e25314c0cc521e743eb39543db37296d4239df46/expr.go#L98-L103) using the value from the lookup.

And that’s everything!
Of course there are a lot of other cases that mingo covers,
but all of those are handled in one or another of the ways described here.
But there’s still one more topic to cover:
the `Analyzer` API.

The `golang.org/x/tools` module defines [a framework](https://pkg.go.dev/golang.org/x/tools/go/analysis) for Go static-analysis tools.
To participate in this framework, an analyzer (like mingo) must present itself as a [analysis.Analyzer](https://pkg.go.dev/golang.org/x/tools/go/analysis#Analyzer).
Doing so allows it to participate in command-line tools based on
[singlechecker](https://pkg.go.dev/golang.org/x/tools/go/analysis/singlechecker)
and [multichecker](https://pkg.go.dev/golang.org/x/tools/go/analysis/multichecker).
(Which isn’t quite good enough for many purposes,
which is why, as of this writing,
there is [work in progress](https://github.com/golang/go/issues/61324) to improve its API.)

Mingo includes an adapter for turning a `Scanner` into an `analysis.Analyzer`,
[here](https://github.com/bobg/mingo/blob/e25314c0cc521e743eb39543db37296d4239df46/analyzer.go#L8).
