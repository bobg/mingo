type eightA struct {
	a, b, c, d, e, f, g, h int
}

type eightB struct {
	a, b, c, d int
	e          int `json:"ee"`
	f, g, h    int
}

var eightC = eightB(eightA{})
