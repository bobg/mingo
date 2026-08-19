type v8_A struct {
	a, b, c, d, e, f, g, h int
}

type v8_B struct {
	a, b, c, d int
	e          int `json:"ee"`
	f, g, h    int
}

var v8_C = v8_B(v8_A{})
