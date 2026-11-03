//go:build !factory_native

package main

import "github.com/levitateos/sodaos/internal/web/serve"

func run() error { return serve.Run(serve.Factory) }
