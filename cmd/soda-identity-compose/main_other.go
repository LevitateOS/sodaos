//go:build !linux

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "soda-identity-compose requires native Linux Project OS")
	os.Exit(1)
}
