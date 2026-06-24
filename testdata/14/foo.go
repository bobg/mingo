type fourteenA interface {
	A()
	B()
}

type fourteenB interface {
	B()
	C()
}

type fourteenC interface {
	fourteenA
	fourteenB
}
