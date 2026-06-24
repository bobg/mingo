type five1 struct {
	a, b int
}

var five2 = map[five1]int{
	{a: 1, b: 2}: 3,
}
