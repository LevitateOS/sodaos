package acceptance

import (
	"bytes"
	_ "embed"
	"errors"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/nativebuild"
)

//go:embed remote_executor.py
var remoteExecutor string

type RemoteRequest struct{ Revision, Architecture, Target, Work, Phase string }

func (r Remote) NativePhase(requestFile, revision, arch, target string) (Command, error) {
	raw, err := PrivateFile(requestFile)
	if err != nil {
		return Command{}, err
	}
	var request RemoteRequest
	if err = nativebuild.ReadJSON(requestFile, &request); err != nil {
		return Command{}, err
	}
	if request.Revision != revision || request.Architecture != arch || request.Target != target {
		return Command{}, errors.New("remote request does not match selected revision/platform/target")
	}
	switch request.Phase {
	case "prepare", "build", "check", "bundle":
	default:
		return Command{}, errors.New("unknown native phase")
	}
	if !filepath.IsAbs(request.Work) || strings.ContainsAny(request.Work, "\r\n\x00") {
		return Command{}, errors.New("absolute remote work path required")
	}
	return r.Command([]string{"python3", "-c", remoteExecutor}, bytes.NewReader(raw))
}
