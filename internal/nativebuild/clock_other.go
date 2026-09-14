//go:build !linux

package nativebuild

import "time"

// Only source tests use this clock; production admission requires native Linux.
var clockOrigin = time.Now()

func monotonic() time.Duration { return time.Since(clockOrigin) }
