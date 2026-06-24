type eighteenD interface {
	int | string
}

func eighteenE[T, U any](t T, u U) (T, U) {
	return t, u
}

var eighteenEIntString = eighteenE[int, string]
