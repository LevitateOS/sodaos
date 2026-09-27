//go:build !linux

package workspace

import "errors"

func (w *Runtime) museArguments(args []string, _ string) ([]string, error) {
	if w.Config.MuseSocket != "" {
		return nil, errors.New("muse worker runtime requires Linux")
	}
	return args, nil
}
