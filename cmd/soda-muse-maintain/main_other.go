//go:build !linux

package main

import (
	"fmt"
	"os"
)

func main() { fmt.Fprintln(os.Stderr, "soda-muse-maintain requires native Linux"); os.Exit(1) }
