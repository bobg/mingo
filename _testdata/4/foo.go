package foo

import "fmt"

func four() {
	var x = []int{1, 2}
	for range x {
		fmt.Println("Almost there...")
	}
}
