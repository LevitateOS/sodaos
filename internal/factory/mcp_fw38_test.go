package factory

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

// FW38 freezes the D14 catalog: seven profiles, fourteen tools, the
// per-profile discovery set and the admission matrix. Unknown tools
// fail; undiscoverable tools still refuse direct calls.
func TestMCPDiscoveryAdmitsReadsEverywhere(t *testing.T) {
	if len(MCPProfiles) != 7 || len(MCPTools) != 14 {
		t.Fatalf("catalog drift: %d profiles, %d tools", len(MCPProfiles), len(MCPTools))
	}
	for _, profile := range MCPProfiles {
		if MCPPhaseOf(profile) == "" {
			t.Fatalf("profile %s serves no phase", profile)
		}
		discovery := MCPDiscovery(profile)
		seen := map[MCPTool]bool{}
		for _, tool := range discovery {
			seen[tool] = true
		}
		for _, read := range []MCPTool{MCPToolContext, MCPToolEvidence, MCPToolStatus, MCPToolReceipt} {
			if !seen[read] {
				t.Fatalf("profile %s misses read %s", profile, read)
			}
		}
	}
}

func TestMCPAdmissionMatrix(t *testing.T) {
	// Typed finishes stay with their role; task output/handoff with
	// builder/fixer; incomplete is the every-parent terminal.
	cases := []struct {
		profile MCPProfile
		tool    MCPTool
		want    bool
	}{
		{MCPProfileProductLead, MCPToolFinishProductPlan, true},
		{MCPProfileProductLead, MCPToolFinishIssuePlan, false},
		{MCPProfileArchitect, MCPToolFinishIssuePlan, true},
		{MCPProfileBuilder, MCPToolFinishImplement, true},
		{MCPProfileBuilder, MCPToolFinishReview, false},
		{MCPProfileInspector, MCPToolFinishReview, true},
		{MCPProfileFixer, MCPToolFinishCorrection, true},
		{MCPProfileFixer, MCPToolFinishImplement, false},
		{MCPProfileStrategist, MCPToolFinishReassessment, true},
		{MCPProfileTestPilot, MCPToolFinishAppVerify, true},
		{MCPProfileTestPilot, MCPToolFinishReassessment, false},
		{MCPProfileBuilder, MCPToolSubmitTaskOutput, true},
		{MCPProfileFixer, MCPToolSubmitTaskOutput, true},
		{MCPProfileInspector, MCPToolSubmitTaskOutput, false},
		{MCPProfileBuilder, MCPToolRequestHandoff, true},
		{MCPProfileFixer, MCPToolRequestHandoff, false},
		{MCPProfileInspector, MCPToolFinishIncomplete, true},
		{MCPProfileTestPilot, MCPToolFinishIncomplete, true},
	}
	for _, c := range cases {
		if got := MCPToolAdmitted(c.profile, c.tool); got != c.want {
			t.Fatalf("admission %s/%s = %v, want %v", c.profile, c.tool, got, c.want)
		}
	}
	// Unknown tools fail server-side for every profile.
	for _, profile := range MCPProfiles {
		if MCPToolAdmitted(profile, "factory.merge") || MCPToolKnown("factory.merge") {
			t.Fatalf("unknown tool admitted for %s", profile)
		}
		for _, tool := range MCPDiscovery(profile) {
			if !MCPToolKnown(tool) {
				t.Fatalf("discovery advertises unknown tool %s", tool)
			}
		}
	}
	if MCPToolKnown("") {
		t.Fatal("empty tool known")
	}
}

func TestMCPEnvelopeBounds(t *testing.T) {
	if err := ValidateMCPEnvelope(nil); err == nil {
		t.Fatal("empty envelope admitted")
	}
	if err := ValidateMCPEnvelope(make([]byte, MCPMaxRequestEnvelopeBytes+1)); err == nil {
		t.Fatal("oversize envelope admitted")
	}
	if err := ValidateMCPEnvelope([]byte(`{}`)); err != nil {
		t.Fatalf("minimal envelope refused: %v", err)
	}
}

func testMCPCapability() MCPCapability {
	digest := strings.Repeat("a", 64)
	return MCPCapability{
		ID: NewID(), AssignmentID: NewID(), RunID: "run-1", Generation: 1,
		Profile: MCPProfileBuilder, Subject: "implement:42", Repository: 7,
		Project:           "p" + strings.Repeat("d", 24),
		PreparationDigest: digest, SourceDigest: "abc", AcceptedRevision: "rev-1",
		Authority: digest, DeadlineUnix: time.Now().Add(time.Hour).Unix(),
		State: MCPCapabilityActive,
	}
}

func TestMCPCapabilityValidation(t *testing.T) {
	if err := testMCPCapability().Validate(); err != nil {
		t.Fatalf("valid capability refused: %v", err)
	}
	bad := testMCPCapability()
	bad.Profile = "coder"
	if err := bad.Validate(); err == nil {
		t.Fatal("unknown profile admitted")
	}
	bad = testMCPCapability()
	bad.State = "live"
	if err := bad.Validate(); err == nil {
		t.Fatal("unknown state admitted")
	}
	bad = testMCPCapability()
	bad.DeadlineUnix = time.Now().Add(-time.Minute).Unix()
	if err := bad.Validate(); err != nil {
		t.Fatalf("expired shape refused at validation (expiry is admission): %v", err)
	}
	if !bad.Expired(time.Now()) {
		t.Fatal("expired capability not detected")
	}
}

func TestMCPCapabilityScopeBindsIdentity(t *testing.T) {
	first := testMCPCapability()
	second := first
	second.Generation = 2
	if first.ScopeDigest() == second.ScopeDigest() {
		t.Fatal("generation change keeps scope digest")
	}
	secret, err := DeriveMCPCapabilitySecret([]byte(strings.Repeat("k", 32)), first)
	if err != nil {
		t.Fatalf("secret derivation refused: %v", err)
	}
	if len(secret) != 64 {
		t.Fatalf("secret is not HMAC-SHA256 hex: %q", secret)
	}
	if _, err := DeriveMCPCapabilitySecret([]byte("short"), first); err == nil {
		t.Fatal("short capability key admitted")
	}
}

func testMCPSubmission(tool MCPTool, payload []byte) MCPSubmission {
	hash := MCPInputDigest(payload)
	return MCPSubmission{
		AssignmentID: NewID(), SubmissionID: NewID(), CapabilityID: NewID(), Tool: tool,
		ExpectedInputDigest: hash, PayloadHash: hash, Payload: payload,
	}
}

func TestMCPSubmissionValidation(t *testing.T) {
	payload := []byte(`{"task_id":"t1"}`)
	if err := testMCPSubmission(MCPToolSubmitTaskOutput, payload).Validate(); err != nil {
		t.Fatalf("valid submission refused: %v", err)
	}
	bad := testMCPSubmission("factory.merge", payload)
	if err := bad.Validate(); err == nil {
		t.Fatal("unknown-tool submission admitted")
	}
	bad = testMCPSubmission(MCPToolStatus, payload)
	bad.PayloadHash = strings.Repeat("b", 64)
	if err := bad.Validate(); err == nil {
		t.Fatal("payload hash mismatch admitted")
	}
	// ExpectedInputDigest is an independent admitted-inputs
	// identity, not a copy of the payload hash: shape keeps both
	// digests well-formed while admission binds the expected digest
	// to the capability's current inputs.
	bad = testMCPSubmission(MCPToolStatus, payload)
	bad.ExpectedInputDigest = strings.Repeat("c", 64)
	if err := bad.Validate(); err != nil {
		t.Fatalf("independent input digest refused at shape: %v", err)
	}
	bad = testMCPSubmission(MCPToolStatus, payload)
	bad.ExpectedInputDigest = "not-a-digest"
	if err := bad.Validate(); err == nil {
		t.Fatal("malformed input digest admitted")
	}
	big := testMCPSubmission(MCPToolStatus, make([]byte, MCPMaxSubmissionPayload+1))
	if err := big.Validate(); err == nil {
		t.Fatal("oversize payload admitted")
	}
	raw := sha256.Sum256(payload)
	if MCPInputDigest(payload) != hex.EncodeToString(raw[:]) {
		t.Fatal("input digest is not SHA-256 hex")
	}
}

func TestMCPReceiptValidation(t *testing.T) {
	receipt := MCPReceipt{
		SubmissionID: NewID(), Tool: MCPToolFinishImplement,
		State: MCPReceiptRecorded, InputDigest: strings.Repeat("d", 64),
	}
	if err := receipt.Validate(); err != nil {
		t.Fatalf("valid receipt refused: %v", err)
	}
	bad := receipt
	bad.State = "pending"
	if err := bad.Validate(); err == nil {
		t.Fatal("unknown receipt state admitted")
	}
	bad = receipt
	bad.Reason = strings.Repeat("r", MCPMaxReasonChars+1)
	if err := bad.Validate(); err == nil {
		t.Fatal("oversize receipt reason admitted")
	}
}

// D16 contract: the proposal covers every accepted outcome with
// approved catalog checks, bounded parameters, artifact outputs and
// fixture requirements. Missing coverage names the outcome.
type stubCatalog map[string]ApprovedCheck

func (s stubCatalog) LookupCheck(id string) (ApprovedCheck, bool) {
	check, ok := s[id]
	return check, ok
}

func testManifestCatalog() stubCatalog {
	return stubCatalog{
		"web.journey.smoke.v3": {ID: "web.journey.smoke.v3", RequiredParams: []string{"journey"}, ParamValueChars: 64},
		"api.check.unit.v1":    {ID: "api.check.unit.v1"},
	}
}

func testManifestProposal() VerificationManifestProposal {
	return VerificationManifestProposal{
		Outcomes: []string{"checkout", "refund"},
		Checks: []ManifestCheckRef{
			{Outcome: "checkout", CatalogID: "web.journey.smoke.v3",
				Params:    map[string]string{"journey": "guest-checkout"},
				Artifacts: []string{"journey-log"}, Fixtures: []string{"demo-store"}},
			{Outcome: "refund", CatalogID: "api.check.unit.v1",
				Artifacts: []string{"unit-report"}},
		},
		FixtureGrants: []string{"demo-store"},
	}
}

func TestValidateManifestProposal(t *testing.T) {
	if err := ValidateManifestProposal(testManifestProposal(), testManifestCatalog()); err != nil {
		t.Fatalf("valid proposal refused: %v", err)
	}
	if got := ManifestCoverage(testManifestProposal()); len(got) != 2 {
		t.Fatalf("coverage = %v, want both outcomes", got)
	}
	if uncovered := ManifestUncovered(testManifestProposal()); len(uncovered) != 0 {
		t.Fatalf("uncovered = %v, want none", uncovered)
	}
}

func TestValidateManifestProposalRefusals(t *testing.T) {
	catalog := testManifestCatalog()
	// Missing outcome coverage.
	missing := testManifestProposal()
	missing.Checks = missing.Checks[:1]
	if err := ValidateManifestProposal(missing, catalog); err == nil {
		t.Fatal("uncovered outcome admitted")
	}
	if uncovered := ManifestUncovered(missing); len(uncovered) != 1 || uncovered[0] != "refund" {
		t.Fatalf("uncovered names %v, want [refund]", uncovered)
	}
	// Unapproved catalog entry.
	unapproved := testManifestProposal()
	unapproved.Checks[0].CatalogID = "model.made.up.v9"
	if err := ValidateManifestProposal(unapproved, catalog); err == nil {
		t.Fatal("unapproved catalog entry admitted")
	}
	// Missing required catalog parameter.
	noparam := testManifestProposal()
	noparam.Checks[0].Params = map[string]string{}
	if err := ValidateManifestProposal(noparam, catalog); err == nil {
		t.Fatal("missing required parameter admitted")
	}
	// Parameter over catalog bound.
	bigparam := testManifestProposal()
	bigparam.Checks[0].Params = map[string]string{"journey": strings.Repeat("j", 65)}
	if err := ValidateManifestProposal(bigparam, catalog); err == nil {
		t.Fatal("over-bound parameter admitted")
	}
	// Check with no artifact output.
	noartifact := testManifestProposal()
	noartifact.Checks[1].Artifacts = nil
	if err := ValidateManifestProposal(noartifact, catalog); err == nil {
		t.Fatal("artifact-less check admitted")
	}
	// Unknown outcome reference.
	unknown := testManifestProposal()
	unknown.Checks[1].Outcome = "chargeback"
	if err := ValidateManifestProposal(unknown, catalog); err == nil {
		t.Fatal("unknown outcome reference admitted")
	}
	// Nil catalog never approves.
	if err := ValidateManifestProposal(testManifestProposal(), nil); err == nil {
		t.Fatal("nil catalog admitted")
	}
}
