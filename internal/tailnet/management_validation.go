package tailnet

import "strings"

func (v EnrollmentView) Validate() error {
	if !validRevision(v.Revision) || v.Tags == nil || v.RuntimeSupported || v.EnrollmentVerified || v.Default {
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
	probe := EnrollmentRequest{Action: "save", Revision: v.Revision, Tailnet: v.Tailnet, Tags: v.Tags, Preauthorized: &v.Preauthorized, ClientID: "validation", ClientSecret: "tskey-client-validation-only"}
	if probe.Validate() != nil {
		return ErrUnavailable
	}
	return nil
}
func (v HostView) Validate() error {
	if (v.Tailnet != "" && !networkPattern.MatchString(v.Tailnet)) || (v.Tailnet == "" && (v.MagicDNSEnabled || v.State == "Running")) {
		return ErrUnavailable
	}
	if !containerPattern.MatchString(v.Revision) || v.Addresses == nil || v.Peers == nil || len(v.Peers) > 128 || v.HealthIssues < 0 || v.HealthIssues > 128 {
		return ErrUnavailable
	}
	switch v.State {
	case "NoState", "InUseOtherUser", "NeedsLogin", "NeedsMachineAuth", "Stopped", "Starting", "Running":
	default:
		return ErrUnavailable
	}
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
	if !validRevision(v.Revision) || v.Available || v.Default {
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
func (v ProjectView) Validate() error {
	if !ValidProject(v.Project) || !validRevision(v.Revision) || (v.Binding != "" && !revisionPattern.MatchString(v.Binding)) || (v.Enabled && v.Binding == "") {
		return ErrUnavailable
	}
	if v.Saved && (v.Enabled || v.Revision == "0" || v.Outcome != "disconnect-unconfirmed") {
		return ErrUnavailable
	}
	if !v.Saved && v.Outcome != "observed" {
		return ErrUnavailable
	}
	if v.State != "runtime-unsupported" || (v.Outcome != "observed" && v.Outcome != "disconnect-unconfirmed") {
		return ErrUnavailable
	}
	return nil
}
