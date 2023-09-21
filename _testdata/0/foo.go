// Nothing in this file
// (variants of code in other subdirs of _testdata)
// should cause an increase in the required Go version.

func zero1() int {
	return 17
}

func zero2() int {
	var x = []int{1, 2}
	for _ = range x {
		println("Almost there...")
	}
	return 0
}

type zero3 struct {
	a, b int
}

var zero4 = map[zero3]int{
	zero3{a: 1, b: 2}: 3,
}

func zero5() int {
	var x = []int{1, 2, 3, 4, 5}
	for _, xx := range x[1:3] {
		println(xx)
	}
	return 0
}

type zero6 struct {
	a, b    int
	c, d    int
	e       int `json:"ee"`
	f, g, h int
}

type zero7 struct {
	a, b, c, d int
	e          int `json:"ee"`
	f, g, h    int
}

var zero8 = zero7(zero6{})

type zero9 int

func zero10() int {
	return 1000
}

var zero11 int = 52 >> uint(2)

func zero12() int {
	x := 1
	x <<= uint(3)
	return x
}

type zero13 interface {
	A()
	B()
}

type zero14 interface {
	C()
	D()
}

type zero15 interface {
	zero13
	zero14
}

func zero16() int {
foo:
	; // labeled empty statement

	ch := make(chan int, 2)
	ch <- 1 // send stmt
	ch <- 2

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

	return 0
}
