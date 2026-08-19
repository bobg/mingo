type v18_D interface {
	int | string
}

func v18_E[T, U any](t T, u U) (T, U) {
	return t, u
}

var v18_EIntString = v18_E[int, string]
