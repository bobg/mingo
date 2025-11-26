import "time"

func seventeen() {
	x := []int{7, 8}
	y := (*[2]int)(x)
	time.Now().GoString()
	println(y)
}
