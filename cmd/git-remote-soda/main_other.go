//go:build !linux

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "Soda Git requires Linux container launch support.")
	os.Exit(1)
}
