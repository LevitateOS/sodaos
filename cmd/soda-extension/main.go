// soda-extension connects Forgejo's private extension transport to Soda.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/web"
	"github.com/levitateos/sodaos/internal/web/api"
)

func run() error {
	flags := flag.NewFlagSet("soda-extension", flag.ContinueOnError)
	packageID := flags.String("package-id", "", "exact package ID: soda or tailnet")
	socket := flags.String("soda-socket", os.Getenv("SODA_EXTENSION_SERVICE_SOCKET"), "absolute path to the private Soda HTTP socket")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || !filepath.IsAbs(*socket) {
		return errors.New("configure an absolute Soda service socket; no positional arguments are accepted")
	}
	application, closeTransport, err := packageApplication(*packageID, *socket)
	if err != nil {
		return err
	}
	defer closeTransport()
	return extensions.Serve(application)
}

func packageApplication(packageID, socket string) (extensions.Application, func(), error) {
	switch packageID {
	case "soda":
		application, closeTransport := web.Extension(socket)
		application.AuthorizeContribution = contributionAuthorizer()
		return application, closeTransport, nil
	case "tailnet":
		handler, closeTransport := api.Extension(socket)
		return extensions.Application{HTTP: handler}, closeTransport, nil
	default:
		return extensions.Application{}, nil, errors.New("package-id must be soda or tailnet")
	}
}

func contributionAuthorizer() extensions.ContributionAuthorizer {
	return func(_ context.Context, request extensions.ContributionRequest) (extensions.ContributionDecision, error) {
		if allowedContribution(request) {
			return extensions.ContributionDecision{Allowed: true}, nil
		}
		return extensions.ContributionDecision{Allowed: false, DenialCode: "contribution_denied"}, nil
	}
}

func allowedContribution(request extensions.ContributionRequest) bool {
	contribution := request.Contribution
	if contribution.Kind == "page" {
		switch contribution.ID {
		case "spaces":
			return contribution.Scope == "global"
		default:
			return false
		}
	}
	return contribution.Kind == "panel" && contribution.Scope == "panel" && contribution.ID == "workspace"
}

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
