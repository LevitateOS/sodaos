// soda-extension connects Forgejo's private extension transport to Soda.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/web"
)

func run() error {
	flags := flag.NewFlagSet("soda-extension", flag.ContinueOnError)
	socket := flags.String("soda-socket", os.Getenv("SODA_EXTENSION_SERVICE_SOCKET"), "absolute path to the private Soda HTTP socket")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || !filepath.IsAbs(*socket) {
		return errors.New("configure an absolute Soda service socket; no positional arguments are accepted")
	}
	application, closeTransport := web.Extension(*socket)
	defer closeTransport()
	return extensions.Serve(application)
}

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
