package tailnet

import (
	"net/netip"
	"net/url"
	"strings"
)

func validEnrollmentIdentity(v EnrollmentView) bool {
	if !validRevision(v.Revision) || v.Tags == nil || v.EnrollmentVerified {
		return false
	}
	return !v.Default || (v.Configured && v.Admission)
}

func emptyEnrollmentBinding(v EnrollmentView) bool {
	return v.Revision == "0" && v.Binding == "" && v.Tailnet == "" && len(v.Tags) == 0
}

func idleEnrollmentFlags(v EnrollmentView) bool {
	return !v.Admission && !v.CredentialChecked && !v.Preauthorized
}

func (v EnrollmentView) Validate() error {
	if !validEnrollmentIdentity(v) {
		return ErrUnavailable
	}
	if !v.Configured {
		if !emptyEnrollmentBinding(v) || !idleEnrollmentFlags(v) {
			return ErrUnavailable
		}
		return nil
	}
	if !revisionPattern.MatchString(v.Revision) || !revisionPattern.MatchString(v.Binding) || !v.CredentialChecked {
		return ErrUnavailable
	}
	if validateEnrollmentPolicy(v.Tailnet, v.Tags, &v.Preauthorized) != nil {
		return ErrUnavailable
	}
	return nil
}

func validHostTailnet(v HostView) bool {
	if v.Tailnet != "" {
		return networkPattern.MatchString(v.Tailnet)
	}
	return !v.MagicDNSEnabled && v.State != "Running"
}

func validHostInventory(v HostView) bool {
	return containerPattern.MatchString(v.Revision) && v.Addresses != nil && v.Peers != nil && len(v.Peers) <= 128 && v.HealthIssues >= 0 && v.HealthIssues <= 128
}

func validHostBackendState(state string) bool {
	switch state {
	case "NoState", "InUseOtherUser", "NeedsLogin", "NeedsMachineAuth", "Stopped", "Starting", "Running":
		return true
	}
	return false
}

func validHostPeers(v HostView) error {
	if validatePeer(v.DNSName, v.Addresses) != nil {
		return ErrUnavailable
	}
	seen := map[string]bool{}
	for _, p := range v.Peers {
		if p.ID == "" || seen[p.ID] || p.Addresses == nil {
			return ErrUnavailable
		}
		seen[p.ID] = true
		if len(p.ID) > 128 || strings.ContainsAny(p.ID, "\r\n\x00") || validatePeer(p.DNSName, p.Addresses) != nil {
			return ErrUnavailable
		}
	}
	return nil
}

func validHostExitNode(v HostView) error {
	if len(v.Preferences.ExitNodeID) > 128 || strings.ContainsAny(v.Preferences.ExitNodeID, "\r\n\x00") {
		return ErrUnavailable
	}
	if v.Preferences.ExitNodeIP != "" {
		if !validTailnetAddresses([]string{v.Preferences.ExitNodeIP}) {
			return ErrUnavailable
		}
	}
	return nil
}

func (v HostView) Validate() error {
	if !validHostTailnet(v) || !validHostInventory(v) || !validHostBackendState(v.State) {
		return ErrUnavailable
	}
	if e := validHostPeers(v); e != nil {
		return e
	}
	return validHostExitNode(v)
}

func (v SettingsView) Validate() error {
	if v.Enrollment.Validate() != nil || (v.Host == nil) != v.HostUnavailable {
		return ErrUnavailable
	}
	if v.Host != nil {
		return v.Host.Validate()
	}
	return nil
}

func (v HostResult) Validate() error {
	switch v.Outcome {
	case "confirmed", "unconfirmed", "pending", "observed":
	default:
		return ErrUnavailable
	}
	if (v.Host == nil) != v.ReadbackUnavailable {
		return ErrUnavailable
	}
	if v.AuthURL != "" && !validAuthenticationURL(v.AuthURL) {
		return ErrUnavailable
	}
	if v.Outcome == "pending" && v.AuthURL == "" {
		return ErrUnavailable
	}
	if v.Host != nil {
		return v.Host.Validate()
	}
	return nil
}

func (v EnrollmentResult) Validate() error {
	if v.Outcome != "confirmed" || !v.CredentialChecked && v.Saved {
		return ErrUnavailable
	}
	return v.Enrollment.Validate()
}

func validProjectOptionsIdentity(v ProjectOptions) bool {
	if v.Revision == "0" {
		return v.Binding == "" && v.Tailnet == ""
	}
	return revisionPattern.MatchString(v.Binding) && networkPattern.MatchString(v.Tailnet)
}

func (v ProjectOptions) Validate() error {
	if !validRevision(v.Revision) || (v.Default && !v.Available) || (v.Available && v.Revision == "0") {
		return ErrUnavailable
	}
	if !validProjectOptionsIdentity(v) {
		return ErrUnavailable
	}
	return nil
}

func validProjectAvailability(binding, network string) bool {
	if (binding == "") != (network == "") {
		return false
	}
	if binding == "" {
		return true
	}
	return revisionPattern.MatchString(binding) && networkPattern.MatchString(network)
}

func validProjectIdentity(v ProjectView) bool {
	return ValidProject(v.Project) && validRevision(v.Revision) && (v.Binding == "" || revisionPattern.MatchString(v.Binding)) && (!v.Enabled || v.Binding != "")
}

func validProjectPersistence(v ProjectView) bool {
	if v.Saved {
		return v.Revision != "0" && (v.Outcome == "disconnect-unconfirmed" || v.Outcome == "runtime-unconfirmed" || v.Outcome == "queued")
	}
	return v.Outcome == "observed"
}

func validDisconnectedProjectState(v ProjectView) bool {
	return len(v.Addresses) == 0 && v.DNSName == ""
}

func validConnectedProjectState(v ProjectView) error {
	if !v.Enabled || len(v.Addresses) == 0 {
		return ErrUnavailable
	}
	if !validPeerDNSName(v.DNSName) || !validTailnetAddresses(v.Addresses) {
		return ErrUnavailable
	}
	return nil
}

func validTailnetAddresses(values []string) bool {
	if len(values) > 16 {
		return false
	}
	for _, value := range values {
		address, err := netip.ParseAddr(value)
		if err != nil || !address.IsGlobalUnicast() || address.String() != value {
			return false
		}
	}
	return true
}

func validPeerDNSName(value string) bool {
	if value == "" {
		return true
	}
	_, err := CanonicalMagicDNSName(value)
	return err == nil
}

func validatePeer(dnsName string, addresses []string) error {
	if !validPeerDNSName(dnsName) || !validTailnetAddresses(addresses) {
		return ErrUnavailable
	}
	return nil
}

func validAuthenticationURL(raw string) bool {
	if len(raw) > 2048 {
		return false
	}
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host == "login.tailscale.com" && u.User == nil && u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && u.RawPath == "" && strings.HasPrefix(u.Path, "/a/") && clientPattern.MatchString(strings.TrimPrefix(u.Path, "/a/")) && u.String() == raw
}

func validProjectRuntime(v ProjectView) error {
	switch v.State {
	case "runtime-unsupported", "off", "stopped", "pending", "needs-login", "approval-required", "unconfirmed":
		if !validDisconnectedProjectState(v) {
			return ErrUnavailable
		}
	case "connected":
		return validConnectedProjectState(v)
	default:
		return ErrUnavailable
	}
	return nil
}

func (v ProjectView) Validate() error {
	if !validProjectAvailability(v.AvailableBinding, v.AvailableNetwork) || !validProjectIdentity(v) || !validProjectPersistence(v) {
		return ErrUnavailable
	}
	return validProjectRuntime(v)
}
