package tailnet

import (
	"slices"
	"strings"
)

// ProjectStatus discards raw health/authentication/peer/private-pref fields. The
// trusted native caller supplies the binding, not the browser. Connected means a
// current node observation; it is not client routing or application authentication.
func ProjectStatus(data, preferences []byte, binding RunBinding) (string, []string, string, error) {
	var s struct {
		Version, BackendState string
		HaveNodeKey           bool
		CurrentTailnet        *struct{ Name string }
		Self                  *struct {
			ID, DNSName        string
			TailscaleIPs, Tags []string
			Online, Expired    bool
		}
	}
	if len(data) > responseLimit || nativeObject(data, &s, "Version", "BackendState", "HaveNodeKey") != nil {
		return "", nil, "", ErrUnavailable
	}
	if strings.SplitN(s.Version, "-", 2)[0] != ManagementCLIRelease {
		return "", nil, "", ErrUnsupported
	}
	switch s.BackendState {
	case "NeedsLogin":
		return "needs-login", nil, "", nil
	case "NeedsMachineAuth":
		return "approval-required", nil, "", nil
	case "Starting", "NoState", "Stopped":
		return "pending", nil, "", nil
	case "Running":
	default:
		return "unconfirmed", nil, "", nil
	}
	var p struct {
		WantRunning, CorpDNS, RouteAll, RunSSH bool
		ExitNodeID, ExitNodeIP                 string
		AdvertiseRoutes                        []string
	}
	if len(preferences) > responseLimit || nativeObject(preferences, &p, "WantRunning", "CorpDNS", "RouteAll", "RunSSH", "ExitNodeID", "ExitNodeIP", "AdvertiseRoutes") != nil {
		return "", nil, "", ErrUnavailable
	}
	if !p.WantRunning || !p.CorpDNS || p.RouteAll || p.RunSSH || p.ExitNodeID != "" || p.ExitNodeIP != "" || len(p.AdvertiseRoutes) != 0 {
		return "", nil, "", ErrConflict
	}
	if !binding.Enabled || s.CurrentTailnet == nil || s.CurrentTailnet.Name != binding.Tailnet || !s.HaveNodeKey || s.Self == nil || s.Self.ID == "" || !s.Self.Online || s.Self.Expired {
		return "unconfirmed", nil, "", nil
	}
	tags := slices.Clone(s.Self.Tags)
	slices.Sort(tags)
	if !slices.Equal(tags, binding.Tags) {
		return "unconfirmed", nil, "", nil
	}
	peer, e := peerView(nativePeer{DNSName: s.Self.DNSName, TailscaleIPs: s.Self.TailscaleIPs})
	if e != nil || len(peer.Addresses) == 0 {
		return "", nil, "", ErrUnavailable
	}
	return "connected", peer.Addresses, peer.DNSName, nil
}
func ProjectCLIRelease(data []byte) error {
	var v struct {
		Short string `json:"short"`
	}
	if nativeObject(data, &v, "short") != nil {
		return ErrUnavailable
	}
	if v.Short != ManagementCLIRelease {
		return ErrUnsupported
	}
	return nil
}

// CheckDaemonRelease permits the pre-login state without claiming enrollment.
func CheckDaemonRelease(data []byte) error {
	var v struct{ Version string }
	if len(data) > responseLimit || nativeObject(data, &v, "Version") != nil {
		return ErrUnavailable
	}
	if strings.SplitN(v.Version, "-", 2)[0] != ManagementCLIRelease {
		return ErrUnsupported
	}
	return nil
}
