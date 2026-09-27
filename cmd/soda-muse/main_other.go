//go:build !linux

package main

import (
	"fmt"
	"os"
)

func main() { fmt.Fprintln(os.Stderr, "Muse launcher requires native Linux"); os.Exit(1) }
