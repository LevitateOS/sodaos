// soda-extension connects Forgejo's private extension transport to Soda.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
	dataDir := extensions.DataDir()
	if !filepath.IsAbs(dataDir) {
		return errors.New("soda extension data directory is unavailable")
	}
	operatorID, err := readOperatorID(filepath.Join(dataDir, "operator-id"))
	if err != nil {
		return err
	}
	application, closeTransport := web.Extension(*socket)
	defer closeTransport()
	application.AuthorizeContribution = contributionAuthorizer(operatorID)
	return extensions.Serve(application)
}

func readOperatorID(path string) (string, error) {
	if !operatorIDFileAvailable(path) {
		return "", errors.New("soda operator identity is unavailable")
	}
	file, err := os.Open(path)
	if err != nil {
		return "", errors.New("soda operator identity is unavailable")
	}
	defer file.Close()
	value, err := io.ReadAll(io.LimitReader(file, 33))
	if err != nil || len(value) == 0 || len(value) > 32 {
		return "", errors.New("soda operator identity is invalid")
	}
	return canonicalOperatorID(value)
}

func operatorIDFileAvailable(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o022 == 0 && info.Size() <= 32
}

func canonicalOperatorID(value []byte) (string, error) {
	operatorID := strings.TrimSuffix(string(value), "\n")
	id, err := strconv.ParseInt(operatorID, 10, 64)
	if err != nil || id < 1 || strconv.FormatInt(id, 10) != operatorID {
		return "", errors.New("soda operator identity is invalid")
	}
	return operatorID, nil
}

func contributionAuthorizer(operatorID string) extensions.ContributionAuthorizer {
	return func(_ context.Context, request extensions.ContributionRequest) (extensions.ContributionDecision, error) {
		if allowedContribution(request, operatorID) {
			return extensions.ContributionDecision{Allowed: true}, nil
		}
		return extensions.ContributionDecision{Allowed: false, DenialCode: "contribution_denied"}, nil
	}
}

func allowedContribution(request extensions.ContributionRequest, operatorID string) bool {
	contribution := request.Contribution
	if contribution.Kind == "page" {
		switch contribution.ID {
		case "spaces":
			return contribution.Scope == "global"
		case "tailnet":
			return contribution.Scope == "admin" && request.ActorID == operatorID
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
