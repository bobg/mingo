type v5_1 struct {
	a, b int
}

var v5_2 = map[v5_1]int{
	{a: 1, b: 2}: 3,
}
