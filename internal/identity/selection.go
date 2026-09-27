package identity

import "errors"

// SelectMuseConnection chooses only current authorized Muse metadata. Multiple
// accounts require an explicit choice rather than an arbitrary subscription.
func SelectMuseConnection(connections []Connection, selected string) (string, error) {
	return selectConnection(connections, Muse, selected, "SODA_MUSE_CONNECTION")
}

func SelectForgejoConnection(connections []Connection, selected string) (string, error) {
	return selectConnection(connections, Forgejo, selected, "SODA_GIT_CONNECTION")
}

func selectConnection(connections []Connection, provider, selected, setting string) (string, error) {
	var matches []string
	for _, connection := range connections {
		if connection.ProviderID != provider || connection.State != Ready {
			continue
		}
		if selected != "" && connection.ID != selected {
			continue
		}
		matches = append(matches, connection.ID)
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return "", errors.New("choose an authorized " + provider + " connection with " + setting)
	}
	return "", ErrDenied
}
