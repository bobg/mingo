type v1_2 struct{}

func (v1_2) x() int {
	return 42
}

func v1_2func() func() int {
	var y v1_2
	return y.x
}
