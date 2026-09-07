//go:build !linux

package acceptance

import "errors"

// Refuse rather than advertise group cleanup without a non-reaping wait.
// Metadata/report operations remain portable; use native Linux for execution.
func ownedGroupsSupported() error {
	return errors.New("safe owned process execution requires Linux non-reaping wait support")
}
func waitOwnedExit(pid int) error { return ownedGroupsSupported() }
