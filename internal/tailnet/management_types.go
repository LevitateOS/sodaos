package tailnet

import (
	"errors"
	"net/netip"
	"regexp"
	"slices"
	"strings"
)

var (
	ErrInvalid        = errors.New("invalid Tailnet request")
	ErrConflict       = errors.New("tailnet revision or identity changed")
	ErrUnsupported    = errors.New("tailnet runtime is not supported")
	ErrUnconfirmed    = errors.New("tailnet outcome is unconfirmed")
	revisionPattern   = regexp.MustCompile(`^[0-9a-f]{32}$`)
	projectPattern    = regexp.MustCompile(`^p[0-9a-f]{24}$`)
	containerPattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	tagPattern        = regexp.MustCompile(`^tag:[a-zA-Z][a-zA-Z0-9-]{0,62}$`)
	credentialPattern = regexp.MustCompile(`^tskey-client-[A-Za-z0-9_-]{8,512}$`) // slop-audit-allow: production validation pattern for real Tailscale-shaped client secrets
	clientPattern     = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
	networkPattern    = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.@_-]{0,252}$`)
)

func validRevision(v string) bool { return v == "0" || revisionPattern.MatchString(v) }
func ValidProject(v string) bool  { return projectPattern.MatchString(v) }

// DTOs deliberately exclude raw native preferences, credential references and
// container/namespace paths. An address is not a reachability receipt.
type HostPreferences struct {
	WantRunning       bool   `json:"want_running"`
	ExitNodeID        string `json:"exit_node_id"`
	ExitNodeIP        string `json:"exit_node_ip"`
	AllowLAN          bool   `json:"allow_lan"`
	AdvertiseExitNode bool   `json:"advertise_exit_node"`
}
type Peer struct {
	ID        string   `json:"id"`
	DNSName   string   `json:"dns_name"`
	Addresses []string `json:"addresses"`
	Online    bool     `json:"online"`
	ExitNode  bool     `json:"exit_node"`
	Expired   bool     `json:"expired"`
}
type HostView struct {
	Tailnet         string          `json:"tailnet"`
	MagicDNSEnabled bool            `json:"magic_dns_enabled"`
	Revision        string          `json:"revision"`
	State           string          `json:"state"`
	HaveNodeKey     bool            `json:"have_node_key"`
	Expired         bool            `json:"expired"`
	DNSName         string          `json:"dns_name"`
	Addresses       []string        `json:"addresses"`
	Peers           []Peer          `json:"peers"`
	HealthIssues    int             `json:"health_issues"`
	Preferences     HostPreferences `json:"preferences"`
}
type EnrollmentView struct {
	Revision           string   `json:"revision"`
	Binding            string   `json:"binding"`
	Tailnet            string   `json:"tailnet"`
	Tags               []string `json:"tags"`
	Configured         bool     `json:"configured"`
	Admission          bool     `json:"admission"`
	Default            bool     `json:"default"`
	Preauthorized      bool     `json:"preauthorized"`
	CredentialChecked  bool     `json:"credential_checked"`
	EnrollmentVerified bool     `json:"enrollment_verified"`
	RuntimeSupported   bool     `json:"runtime_supported"`
}
type SettingsView struct {
	Host            *HostView      `json:"host"`
	HostUnavailable bool           `json:"host_unavailable"`
	Enrollment      EnrollmentView `json:"enrollment"`
}
type HostRequest struct {
	Action    string  `json:"action"`
	Revision  string  `json:"revision"`
	Confirm   string  `json:"confirm,omitempty"`
	ExitNode  *string `json:"exit_node,omitempty"`
	AllowLAN  *bool   `json:"allow_lan,omitempty"`
	Advertise *bool   `json:"advertise,omitempty"`
}

func hasExtraHostFields(r HostRequest) bool {
	return r.ExitNode != nil || r.AllowLAN != nil || r.Advertise != nil
}

func validateHostSigninAction(r HostRequest) error {
	if r.Confirm != "" || hasExtraHostFields(r) {
		return ErrInvalid
	}
	return nil
}

func validateHostConfirmedAction(r HostRequest) error {
	if r.Confirm != r.Action || hasExtraHostFields(r) {
		return ErrInvalid
	}
	return nil
}

func validateHostExitNode(exitNode *string, allowLAN *bool) error {
	if *exitNode == "" {
		if *allowLAN {
			return ErrInvalid
		}
		return nil
	}
	a, err := netip.ParseAddr(*exitNode)
	if err != nil || a.String() != *exitNode || !a.IsGlobalUnicast() {
		return ErrInvalid
	}
	return nil
}

func validateHostExitNodeAction(r HostRequest) error {
	if r.Confirm != r.Action || r.ExitNode == nil || r.AllowLAN == nil || r.Advertise != nil {
		return ErrInvalid
	}
	return validateHostExitNode(r.ExitNode, r.AllowLAN)
}

func validateHostAdvertiseAction(r HostRequest) error {
	if r.Confirm != r.Action || r.Advertise == nil || r.ExitNode != nil || r.AllowLAN != nil {
		return ErrInvalid
	}
	return nil
}

func (r HostRequest) Validate() error {
	if !containerPattern.MatchString(r.Revision) {
		return ErrInvalid
	}
	switch r.Action {
	case "signin", "authentication":
		return validateHostSigninAction(r)
	case "logout", "refresh-forgejo":
		return validateHostConfirmedAction(r)
	case "exit-node":
		return validateHostExitNodeAction(r)
	case "advertise-exit-node":
		return validateHostAdvertiseAction(r)
	default:
		return ErrInvalid
	}
}

type HostResult struct {
	Outcome             string    `json:"outcome"`
	Host                *HostView `json:"host"`
	ReadbackUnavailable bool      `json:"readback_unavailable"`
	AuthURL             string    `json:"auth_url,omitempty"`
}
type EnrollmentRequest struct {
	Action        string   `json:"action"`
	Revision      string   `json:"revision"`
	Tailnet       string   `json:"tailnet,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Preauthorized *bool    `json:"preauthorized,omitempty"`
	ClientID      string   `json:"client_id,omitempty"`
	ClientSecret  string   `json:"client_secret,omitempty"`
	Default       *bool    `json:"default,omitempty"`
}

func validateEnrollmentTags(tags []string) error {
	if len(tags) < 1 || len(tags) > 8 || !slices.IsSorted(tags) {
		return ErrInvalid
	}
	for i, t := range tags {
		if !tagPattern.MatchString(t) || (i > 0 && tags[i-1] == t) {
			return ErrInvalid
		}
	}
	return nil
}

func validateEnrollmentPolicy(tailnet string, tags []string, preauthorized *bool) error {
	if !networkPattern.MatchString(tailnet) || strings.Contains(tailnet, "..") || preauthorized == nil {
		return ErrInvalid
	}
	return validateEnrollmentTags(tags)
}

func validateEnrollmentMutation(r EnrollmentRequest) error {
	if !clientPattern.MatchString(r.ClientID) || !credentialPattern.MatchString(r.ClientSecret) || r.Default != nil {
		return ErrInvalid
	}
	return validateEnrollmentPolicy(r.Tailnet, r.Tags, r.Preauthorized)
}

func hasEnrollmentPayload(r EnrollmentRequest) bool {
	return r.ClientID != "" || r.ClientSecret != "" || r.Tailnet != "" || r.Tags != nil || r.Preauthorized != nil
}

func validateEnrollmentToggle(r EnrollmentRequest) error {
	if hasEnrollmentPayload(r) {
		return ErrInvalid
	}
	if r.Action == "default" && r.Default == nil {
		return ErrInvalid
	}
	if r.Action == "disable" && r.Default != nil {
		return ErrInvalid
	}
	return nil
}

func (r EnrollmentRequest) Validate() error {
	if !validRevision(r.Revision) {
		return ErrInvalid
	}
	switch r.Action {
	case "save", "check", "rotate":
		return validateEnrollmentMutation(r)
	case "default", "disable":
		return validateEnrollmentToggle(r)
	default:
		return ErrInvalid
	}
}

type EnrollmentResult struct {
	Outcome           string         `json:"outcome"`
	Saved             bool           `json:"saved"`
	CredentialChecked bool           `json:"credential_checked"`
	Enrollment        EnrollmentView `json:"enrollment"`
}
type ProjectOptions struct {
	Revision  string `json:"revision"`
	Binding   string `json:"binding"`
	Tailnet   string `json:"tailnet"`
	Available bool   `json:"available"`
	Default   bool   `json:"default"`
}
type ProjectRequest struct {
	Project   string `json:"project"`
	Action    string `json:"action"`
	Revision  string `json:"revision,omitempty"`
	Binding   string `json:"binding,omitempty"`
	ConfirmID string `json:"confirm_id,omitempty"`
}

func validateProjectMutation(r ProjectRequest) error {
	if !validRevision(r.Revision) || r.ConfirmID != r.Project {
		return ErrInvalid
	}
	switch r.Action {
	case "disable":
		if r.Binding != "" {
			return ErrInvalid
		}
	case "enable", "retry":
		if !revisionPattern.MatchString(r.Binding) {
			return ErrInvalid
		}
	default:
		return ErrInvalid
	}
	return nil
}

func (r ProjectRequest) Validate() error {
	if !ValidProject(r.Project) {
		return ErrInvalid
	}
	if r.Action == "inspect" {
		if r.Revision != "" || r.Binding != "" || r.ConfirmID != "" {
			return ErrInvalid
		}
		return nil
	}
	return validateProjectMutation(r)
}

type ProjectView struct {
	AvailableBinding string   `json:"available_binding,omitempty"`
	AvailableNetwork string   `json:"available_network,omitempty"`
	Addresses        []string `json:"addresses,omitempty"`
	DNSName          string   `json:"dns_name,omitempty"`
	Saved            bool     `json:"saved"`
	Project          string   `json:"project"`
	Revision         string   `json:"revision"`
	Binding          string   `json:"binding"`
	Enabled          bool     `json:"enabled"`
	State            string   `json:"state"`
	Outcome          string   `json:"outcome"`
}
