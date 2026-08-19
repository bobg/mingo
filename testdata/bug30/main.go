package main

import "context"

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	ctx := context.Background()
	return run2(ctx)
}

func run2(ctx context.Context) error {
	return ctx.Err()
}
