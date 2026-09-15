package tailnet

import "slices"

// ProjectStatus discards raw health/authentication/peer/private-pref fields. The
// trusted native caller supplies the binding, not the browser. Connected means a
// current node observation; it is not client routing or application authentication.
type projectNativeStatus struct {
	BackendState   string
	HaveNodeKey    bool
	CurrentTailnet *struct{ Name string }
	Self           *struct {
		ID, DNSName        string
		TailscaleIPs, Tags []string
		Online, Expired    bool
	}
}

type projectNativePrefs struct {
	WantRunning, CorpDNS, RouteAll, RunSSH bool
	ExitNodeID, ExitNodeIP                 string
	AdvertiseRoutes                        []string
}

func parseProjectStatus(data []byte) (projectNativeStatus, string, error) {
	var s projectNativeStatus
	if len(data) > responseLimit || nativeObject(data, &s, "BackendState", "HaveNodeKey") != nil {
		return s, "", ErrUnavailable
	}
	switch s.BackendState {
	case "NeedsLogin":
		return s, "needs-login", nil
	case "NeedsMachineAuth":
		return s, "approval-required", nil
	case "Starting", "NoState", "Stopped":
		return s, "pending", nil
	case "Running":
		return s, "", nil
	default:
		return s, "unconfirmed", nil
	}
}

func invalidProjectPrefs(p projectNativePrefs) bool {
	if !p.WantRunning || !p.CorpDNS || p.RouteAll || p.RunSSH {
		return true
	}
	return p.ExitNodeID != "" || p.ExitNodeIP != "" || len(p.AdvertiseRoutes) != 0
}

func validateProjectPreferences(preferences []byte) error {
	var p projectNativePrefs
	if len(preferences) > responseLimit || nativeObject(preferences, &p, "WantRunning", "CorpDNS", "RouteAll", "RunSSH", "ExitNodeID", "ExitNodeIP", "AdvertiseRoutes") != nil {
		return ErrUnavailable
	}
	if invalidProjectPrefs(p) {
		return ErrConflict
	}
	return nil
}

func matchProjectSelf(self *struct {
	ID, DNSName        string
	TailscaleIPs, Tags []string
	Online, Expired    bool
}, tags []string,
) bool {
	if self == nil || self.ID == "" || !self.Online || self.Expired {
		return false
	}
	cloned := slices.Clone(self.Tags)
	slices.Sort(cloned)
	return slices.Equal(cloned, tags)
}

func matchProjectBinding(s projectNativeStatus, binding RunBinding) bool {
	if !binding.Enabled || s.CurrentTailnet == nil || s.CurrentTailnet.Name != binding.Tailnet || !s.HaveNodeKey {
		return false
	}
	return matchProjectSelf(s.Self, binding.Tags)
}

func resolveProjectPeer(dnsName string, ips []string) ([]string, string, error) {
	peer, err := peerView(nativePeer{DNSName: dnsName, TailscaleIPs: ips})
	if err != nil || len(peer.Addresses) == 0 {
		return nil, "", ErrUnavailable
	}
	return peer.Addresses, peer.DNSName, nil
}

// ProjectStatus discards raw health/authentication/peer/private-pref fields. The
// trusted native caller supplies the binding, not the browser. Connected means a
// current node observation; it is not client routing or application authentication.
func ProjectStatus(data, preferences []byte, binding RunBinding) (string, []string, string, error) {
	s, outcome, err := parseProjectStatus(data)
	if err != nil {
		return "", nil, "", err
	}
	if outcome != "" {
		return outcome, nil, "", nil
	}
	if err := validateProjectPreferences(preferences); err != nil {
		return "", nil, "", err
	}
	if !matchProjectBinding(s, binding) {
		return "unconfirmed", nil, "", nil
	}
	addrs, dnsName, err := resolveProjectPeer(s.Self.DNSName, s.Self.TailscaleIPs)
	if err != nil {
		return "", nil, "", err
	}
	return "connected", addrs, dnsName, nil
}

// ProjectHasNode admits a concrete pre-login observation, not a release number.
// An existing key (including approval/reauthentication states) must not be replaced
// by another enrollment. Missing or unfamiliar state is unavailable, not absence.
func ProjectHasNode(data []byte) (bool, error) {
	var v struct {
		BackendState string
		HaveNodeKey  bool
	}
	if len(data) > responseLimit || nativeObject(data, &v, "BackendState", "HaveNodeKey") != nil {
		return false, ErrUnavailable
	}
	switch v.BackendState {
	case "NoState", "NeedsLogin", "NeedsMachineAuth", "Stopped", "Starting":
	case "Running":
		if !v.HaveNodeKey {
			return false, ErrUnavailable
		}
	default:
		return false, ErrUnavailable
	}
	return v.HaveNodeKey, nil
}
