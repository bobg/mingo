type twentysevenA struct {
	twentysevenB
	b int
}

type twentysevenB struct {
	c string
	d bool
}

var twentysevenC = twentysevenA{
	c: "hello",
}
