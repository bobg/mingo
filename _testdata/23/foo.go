func twentythree() {
	f := func(yield func(int) bool) {
		for n := 0; n < 10; n++ {
			if !yield(n) {
				return
			}
		}
	}

	for x := range f {
		println(x)
	}
}
