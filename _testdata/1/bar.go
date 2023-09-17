type one2 struct{}

func (one2) x() {}

func one2func() func() {
	var y one2
	return y.x
}
