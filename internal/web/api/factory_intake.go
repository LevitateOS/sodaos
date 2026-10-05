package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	hostpublish "github.com/levitateos/sodaos/internal/host/publish"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// IntakeCoordinator consumes one authenticated native event hint into a
// readiness assessment. The coordinator implements it; tests supply fakes.
type IntakeCoordinator interface {
	ObserveIssueEvent(ctx context.Context, hint control.IntakeHint) (factory.IssueControl, bool, error)
}

// IntakeHandler serves authenticated native webhook deliveries as
// readiness event hints. Fountain signs each delivery with the shared
// intake secret; the handler verifies the HMAC-SHA256 hex signature,
// extracts the affected issue locator, and lets the coordinator assess
// from authoritative snapshots. Hints are partial by design: unknown
// fields and unrelated events are ignored, never refused.
type IntakeHandler struct {
	Coordinator IntakeCoordinator
	Secret      []byte
}

func intakeHeader(r *http.Request, forgejo, gitea string) string {
	if value := r.Header.Get(forgejo); value != "" {
		return value
	}
	return r.Header.Get(gitea)
}

type intakeOutcome struct {
	Repository string `json:"repository,omitempty"`
	Issue      string `json:"issue,omitempty"`
	Readiness  string `json:"readiness,omitempty"`
	Changed    bool   `json:"changed,omitempty"`
	Ignored    bool   `json:"ignored,omitempty"`
}

func (h IntakeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		auth.JSONError(w, http.StatusMethodNotAllowed, "invalid_method", "Factory intake accepts POST deliveries.")
		return
	}
	if len(h.Secret) == 0 || h.Coordinator == nil {
		auth.JSONError(w, http.StatusServiceUnavailable, "intake_unavailable", "Factory intake is not configured.")
		return
	}
	delivery := intakeHeader(r, "X-Forgejo-Delivery", "X-Gitea-Delivery")
	if delivery == "" {
		auth.JSONError(w, http.StatusBadRequest, "invalid_delivery", "Native delivery identity is required.")
		return
	}
	signature := intakeHeader(r, "X-Forgejo-Signature", "X-Gitea-Signature")
	body, err := io.ReadAll(io.LimitReader(r.Body, int64(auth.APIBodyLimit)+1))
	if err != nil || len(body) > auth.APIBodyLimit {
		auth.JSONError(w, http.StatusRequestEntityTooLarge, "delivery_too_large", "Native delivery exceeds the intake limit.")
		return
	}
	if !verifyIntakeSignature(h.Secret, signature, body) {
		auth.JSONError(w, http.StatusUnauthorized, "invalid_signature", "Native delivery signature is invalid.")
		return
	}
	event := intakeHeader(r, "X-Forgejo-Event", "X-Gitea-Event")
	hint, ignored, ok := parseIntakeHint(w, event, delivery, body)
	if !ok {
		return
	}
	if ignored {
		auth.JSONResponse(w, http.StatusOK, intakeOutcome{Ignored: true})
		return
	}
	if err := hint.Validate(); err != nil {
		auth.JSONError(w, http.StatusBadRequest, "invalid_hint", "Native delivery does not name a factory issue.")
		return
	}
	assessed, changed, err := h.Coordinator.ObserveIssueEvent(r.Context(), hint)
	if err != nil {
		attrs := []any{"delivery", delivery, "event", event, "error", err}
		var status *hostpublish.StatusError
		if errors.As(err, &status) {
			attrs = append(attrs, "native_status", status.Status, "native_body", status.Body)
		}
		slog.Warn("factory intake assessment unavailable", attrs...)
		auth.JSONError(w, http.StatusServiceUnavailable, "intake_unavailable", "Readiness assessment is unavailable; the native side retries.")
		return
	}
	if assessed.Readiness == "" {
		auth.JSONResponse(w, http.StatusOK, intakeOutcome{Ignored: true})
		return
	}
	auth.JSONResponse(w, http.StatusOK, intakeOutcome{
		Repository: strconv.FormatInt(assessed.Repository, 10),
		Issue:      strconv.FormatInt(assessed.Issue, 10),
		Readiness:  assessed.Readiness, Changed: changed,
	})
}

// verifyIntakeSignature compares the hex HMAC-SHA256 of the raw delivery
// body against the native signature header in constant time.
func verifyIntakeSignature(secret []byte, signature string, body []byte) bool {
	if signature == "" {
		return false
	}
	want, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	sum := hmac.New(sha256.New, secret)
	_, _ = sum.Write(body)
	return subtle.ConstantTimeCompare(sum.Sum(nil), want) == 1
}

type intakePermissions struct {
	Admin bool `json:"admin"`
	Push  bool `json:"push"`
	Pull  bool `json:"pull"`
}

type intakeRepository struct {
	ID          int64              `json:"id"`
	Permissions *intakePermissions `json:"permissions"`
}

type intakeSender struct {
	ID int64 `json:"id"`
}

type intakeIssue struct {
	Index int64 `json:"index"`
}

type intakeIssueEvent struct {
	Issue      *intakeIssue      `json:"issue"`
	Repository *intakeRepository `json:"repository"`
	Sender     *intakeSender     `json:"sender"`
	Action     string            `json:"action"`
	Index      int64             `json:"number"`
}

type intakeCommentEvent struct {
	Issue      *intakeIssue      `json:"issue"`
	Repository *intakeRepository `json:"repository"`
	IsPull     bool              `json:"is_pull"`
}

// parseIntakeHint extracts the affected issue from one verified delivery.
// Issue and comment events map to hints; pull requests and unrelated
// events are ignored. Creation observations additionally carry the sender
// and their code-write authority at event build time; the coordinator
// re-verifies the creator against authoritative provenance before any
// adoption.
func parseIntakeHint(w http.ResponseWriter, event, delivery string, body []byte) (control.IntakeHint, bool, bool) {
	bad := func() (control.IntakeHint, bool, bool) {
		auth.JSONError(w, http.StatusBadRequest, "invalid_hint", "Native delivery does not name a factory issue.")
		return control.IntakeHint{}, false, false
	}
	ignored := func() (control.IntakeHint, bool, bool) {
		return control.IntakeHint{}, true, true
	}
	switch event {
	case "issues":
		var payload intakeIssueEvent
		if json.Unmarshal(body, &payload) != nil || payload.Repository == nil {
			return bad()
		}
		if payload.Index <= 0 || payload.Repository.ID <= 0 {
			return bad()
		}
		if payload.Issue != nil && payload.Issue.Index != payload.Index {
			return bad()
		}
		hint := control.IntakeHint{
			Delivery: delivery, Repository: payload.Repository.ID, Issue: payload.Index,
		}
		if payload.Action == "opened" {
			if payload.Sender == nil || payload.Sender.ID <= 0 {
				return bad()
			}
			hint.Created = true
			hint.Creator = payload.Sender.ID
			if payload.Repository.Permissions != nil {
				hint.CreatorWrite = payload.Repository.Permissions.Push || payload.Repository.Permissions.Admin
			}
		}
		return hint, false, true
	case "issue_comment":
		var payload intakeCommentEvent
		if json.Unmarshal(body, &payload) != nil || payload.Repository == nil || payload.Issue == nil {
			return bad()
		}
		if payload.IsPull {
			return ignored()
		}
		if payload.Issue.Index <= 0 || payload.Repository.ID <= 0 {
			return bad()
		}
		return control.IntakeHint{
			Delivery: delivery, Repository: payload.Repository.ID, Issue: payload.Issue.Index,
		}, false, true
	default:
		return ignored()
	}
}
