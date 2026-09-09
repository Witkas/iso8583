//go:build !js || !wasm

// This stub lets `go build ./...` and `go vet ./...` succeed on non-wasm
// platforms, where main.go (js && wasm only) is excluded. The real entry point
// is built with: GOOS=js GOARCH=wasm go build -o iso8583.wasm ./cmd/wasm
package main

import "fmt"

func main() {
	fmt.Println("Build the browser decoder with: GOOS=js GOARCH=wasm go build -o iso8583.wasm ./cmd/wasm")
}
