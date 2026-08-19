type v27_D string

func (d v27_D) ifHello[T any](val T) T {
	if string(d) == "hello" {
		return val
	}
	var res T
	return res
}
