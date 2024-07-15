package main

import "github.com/malivvan/zero"

func main() {
	if err := zero.Run(zero.Config{
		NonBlockingStdio: true,
	}, "app.wasm", []string{}); err != nil {
		panic(err)
	}
}
