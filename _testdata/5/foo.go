package foo

import "fmt"

func one() int {
	if true {
		return 17
	}
}

func four() {
	var x = []int{1, 2}
	for range x {
		fmt.Println("Almost there...")
	}
}

type five struct {
	a, b int
}

var x = map[five]int{
	{a: 1, b: 2}: 3,
}
