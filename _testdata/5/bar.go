func five2() {
	var x = []int{1, 2, 3, 4, 5}
	for _, xx := range x[1:3:5] {
		println(xx)
	}
}
