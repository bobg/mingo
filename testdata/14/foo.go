type v14_A interface {
	A()
	B()
}

type v14_B interface {
	B()
	C()
}

type v14_C interface {
	v14_A
	v14_B
}
