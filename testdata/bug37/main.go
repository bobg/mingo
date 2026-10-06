package main

import "os"

func main() {
	r, _ := os.OpenRoot(".")

	if err := r.MkdirAll("a", 0o755); err != nil {
		panic(err)
	}
}
