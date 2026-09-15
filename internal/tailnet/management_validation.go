package tailnet

import "strings"

func (v EnrollmentView) Validate() error {
	if !validRevision(v.Revision) || v.Tags == nil || v.EnrollmentVerified || (v.Default && (!v.Configured || !v.Admission)) {
		return ErrUnavailable
	}
	if !v.Configured {
		if v.Revision != "0" || v.Binding != "" || v.Tailnet != "" || len(v.Tags) != 0 || v.Admission || v.CredentialChecked || v.Preauthorized {
			return ErrUnavailable
		}
		return nil
	}
	if !revisionPattern.MatchString(v.Revision) || !revisionPattern.MatchString(v.Binding) || !v.CredentialChecked {
		return ErrUnavailable
	}
	probe := EnrollmentRequest{Action: "save", Revision: v.Revision, Tailnet: v.Tailnet, Tags: v.Tags, Preauthorized: &v.Preauthorized, ClientID: "validation", ClientSecret: "tskey-client-validation-only"} // slop-audit-allow: synthetic probe that must pass credentialPattern to exercise the real validation path
	if probe.Validate() != nil {
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
	if _, e := peerView(nativePeer{DNSName: v.DNSName, TailscaleIPs: v.Addresses}); e != nil {
		return e
	}
	seen := map[string]bool{}
	for _, p := range v.Peers {
		if p.ID == "" || seen[p.ID] || p.Addresses == nil {
			return ErrUnavailable
		}
		seen[p.ID] = true
		if _, e := peerView(nativePeer{ID: p.ID, DNSName: p.DNSName, TailscaleIPs: p.Addresses}); e != nil {
			return e
		}
	}
	return nil
}

func validHostExitNode(v HostView) error {
	if len(v.Preferences.ExitNodeID) > 128 || strings.ContainsAny(v.Preferences.ExitNodeID, "\r\n\x00") {
		return ErrUnavailable
	}
	if v.Preferences.ExitNodeIP != "" {
		if _, e := addresses([]string{v.Preferences.ExitNodeIP}); e != nil {
			return e
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
	if v.AuthURL != "" && authenticationURL(v.AuthURL) != v.AuthURL {
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

func (v ProjectOptions) Validate() error {
	if !validRevision(v.Revision) || (v.Default && !v.Available) || (v.Available && v.Revision == "0") {
		return ErrUnavailable
	}
	if v.Revision == "0" {
		if v.Binding != "" || v.Tailnet != "" {
			return ErrUnavailable
		}
	} else if !revisionPattern.MatchString(v.Binding) || !networkPattern.MatchString(v.Tailnet) {
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
	if _, e := peerView(nativePeer{DNSName: v.DNSName, TailscaleIPs: v.Addresses}); e != nil {
		return e
	}
	return nil
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
