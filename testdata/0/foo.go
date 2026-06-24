// Nothing in this file
// (variants of code in other subdirs of testdata)
// should cause an increase in the required Go version.

func v0_1() int {
	return 17
}

func v0_2() int {
	var x = []int{1, 2}
	for _ = range x {
		println("Almost there...")
	}
	return 0
}

type v0_3 struct {
	a, b int
}

var v0_4 = map[v0_3]int{
	v0_3{a: 1, b: 2}: 3,
}

func v0_5() int {
	var x = []int{1, 2, 3, 4, 5}
	for _, xx := range x[1:3] {
		println(xx)
	}
	return 0
}

type v0_6 struct {
	a, b    int
	c, d    int
	e       int `json:"ee"`
	f, g, h int
}

type v0_7 struct {
	a, b, c, d int
	e          int `json:"ee"`
	f, g, h    int
}

var v0_8 = v0_7(v0_6{})

type v0_9 int

func v0_10() int {
	return 1000
}

var v0_11 int = 52 >> uint(2)

func v0_12() int {
	x := 1
	x <<= uint(3)
	return x
}

type v0_13 interface {
	A()
	B()
}

type v0_14 interface {
	C()
	D()
}

type v0_15 interface {
	v0_13
	v0_14
}

func v0_16() int {
foo:
	; // labeled empty statement

	nums := []int{0, 1, 2}

	ch := make(chan int, 2)
	ch <- nums[1]   // send stmt, index expr
	ch <- (nums[2]) // paren expr

	x := <-ch
	x++ // incdec stmt

	// go stmt
	go func() {
		println("heyo")
		return
	}()

	defer println("deferred") // defer stmt

	for {
		break // branch stmt
	}

	if x < 1 {
		goto foo
	} else {
		println("goto considered harmful")
	}

	switch x {
	case 2:
		println("thought so")
	}

	select {
	case y := <-ch:
		println(y)
	}

	var z interface{} = 1
	switch z.(type) {
	case int:
		println("z is int")
	default:
		println("z is not int")
	}

	zptr := &z
	println(*zptr) // star expr

	return 0
}

func v0_17(x int) int {
	return x + 1
}

var v0_18 = v0_17(1)
