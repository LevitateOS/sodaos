package main

import (
	"log/slog"
	"os"

	"github.com/levitateos/sodaos/internal/web/serve"
)

func main() {
	if err := serve.Run(serve.Forge); err != nil {
		slog.Error("forge stopped", "error", err)
		os.Exit(1)
	}
}
