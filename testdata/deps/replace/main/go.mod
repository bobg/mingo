module example.com/main

go 1.26

require (
	example.com/direct v1.0.0
	example.com/indirect v1.0.0 // indirect
	example.com/tagdep v1.0.0
	example.com/testdep v1.0.0
)

replace (
	example.com/direct => ../direct
	example.com/indirect => ../indirect
	example.com/tagdep => ../tagdep
	example.com/testdep => ../testdep
)
