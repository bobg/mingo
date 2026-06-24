type nineA = int

func nineB(x interface{}) interface{} {
	type any = interface{}

	var y any = x // should not require Go 1.18 (where any is predefined)
	return y
}
