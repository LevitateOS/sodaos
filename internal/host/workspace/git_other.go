//go:build !linux

package workspace

import "errors"

func (w *Runtime) gitArguments(args []string) ([]string, error) {
	if w.Config.GitSocket != "" {
		return nil, errors.New("factory Git runtime requires Linux")
	}
	return args, nil
}
