type one2 struct{}

func (one2) x() int {
	return 42
}

func one2func() func() int {
	var y one2
	return y.x
}
