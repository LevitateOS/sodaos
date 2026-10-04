package installer

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// enrollmentSELinuxPortType labels the temporary import listener as what it
// is: a genuine OpenSSH service. The bind runs in a context that may use
// ssh_port_t, while the default unreserved_port_t bind is denied.
const enrollmentSELinuxPortType = "ssh_port_t"

// enrollmentPortLabeled reports whether a semanage port listing already
// assigns the enrollment port to the SSH port type.
func enrollmentPortLabeled(listing []byte) bool {
	for _, line := range strings.Split(string(listing), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[0] != enrollmentSELinuxPortType || fields[1] != "tcp" {
			continue
		}
		for _, port := range strings.Split(strings.Join(fields[2:], " "), ",") {
			if strings.TrimSpace(port) == enrollmentPort {
				return true
			}
		}
	}
	return false
}

// ensureEnrollmentPortLabel makes the enrollment bind survivable where
// SELinux enforces. Non-enforcing systems skip silently: labeling is
// meaningless without enforcement, and semanage may not exist there.
func ensureEnrollmentPortLabel(ctx context.Context, run commandRunner, enforcing bool) error {
	if !enforcing {
		return nil
	}
	listing, err := run(ctx, "semanage", []string{"port", "-l"}, nil)
	if err != nil {
		return errors.New("SELinux is enforcing but the SSH port labeling cannot be inspected; install policycoreutils-python-utils or label tcp/" + enrollmentPort + " " + enrollmentSELinuxPortType + " manually")
	}
	if enrollmentPortLabeled(listing) {
		return nil
	}
	if _, err := run(ctx, "semanage", []string{"port", "-a", "-t", enrollmentSELinuxPortType, "-p", "tcp", enrollmentPort}, nil); err != nil {
		return fmt.Errorf("label tcp/%s %s: %w", enrollmentPort, enrollmentSELinuxPortType, err)
	}
	return nil
}

// labelEnrollmentConfig lets the per-connection sshd read its config. The
// state directory carries var_run_t, which sshd cannot read; mirror the
// native /etc/ssh/sshd_config type instead.
func labelEnrollmentConfig(ctx context.Context, run commandRunner, enforcing bool) error {
	if _, err := run(ctx, "chcon", []string{"-t", "etc_t", enrollmentConfigPath}, nil); err != nil {
		if !enforcing {
			return nil
		}
		return fmt.Errorf("label enrollment SSH config: %w", err)
	}
	return nil
}
