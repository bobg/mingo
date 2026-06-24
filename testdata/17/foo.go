func seventeen() {
	x := []int{7, 8}
	y := (*[2]int)(x)
	println(y)
}
