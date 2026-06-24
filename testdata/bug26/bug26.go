// Thank you to Matthew R Kasun for reporting this issue!
// See https://github.com/bobg/mingo/issues/26

package main

import (
	"fmt"
	"text/template/parse"
)

func main() {
	foo := parse.BreakNode{}
	fmt.Print(foo)
}
