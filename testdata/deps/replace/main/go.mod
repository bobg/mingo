module example.com/main

go 1.25

require (
	example.com/direct v1.0.0
	example.com/indirect v1.0.0 // indirect
	example.com/testdep v1.0.0
)

replace (
	example.com/direct => ../direct
	example.com/indirect => ../indirect
	example.com/testdep => ../testdep
)
