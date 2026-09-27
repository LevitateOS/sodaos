//go:build !linux

package main

import "errors"

func workerRuntimeAvailable(string) error { return errors.New("muse workers require native Linux") }
