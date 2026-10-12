package factory

import (
	"encoding/json"
	"strings"
	"testing"
)

// hflowCurrent is the strict H-FLOW wire example: a current saved-review
// read. Every review payload binds the basis revision the round composed
// from; the head still is the result revision saved by that round.
const hflowCurrent = `{
  "intent": "d000000000000000000000000",
  "head_revision": "d222222222222222222222222",
  "status": "current",
  "review": {
    "intent": "d000000000000000000000000",
    "basis_revision": "d111111111111111111111111",
    "result_revision": "d222222222222222222222222",
    "content": {
      "producer": "soda-authored-v1",
      "supported": true,
      "intent": "d000000000000000000000000",
      "revision": "d111111111111111111111111",
      "objective": "Time-off requests my manager can approve",
      "facts": ["Time-off requests my manager can approve"],
      "alternatives": [
        {
          "key": "single_person",
          "name": "Small single-person workflow",
          "audience": "One person completing their own request without a separate decider.",
          "summary": "The person records what they need and sees the recorded result directly.",
          "steps": [
            {"actor": "The person", "action": "describes what they need", "result": "The request is recorded"},
            {"actor": "The person", "action": "confirms the recorded request", "result": "The request counts as submitted"}
          ],
          "visibility": "Only the person sees their own request unless the description says otherwise.",
          "consequences": "No waiting on another person, and no second check before the request counts.",
          "evidence": "Authored pattern; no external research was performed.",
          "outcomes": ["The person completes a time-off request"]
        },
        {
          "key": "shared_review",
          "name": "Shared review workflow",
          "audience": "A person making a request and another person deciding on it.",
          "summary": "The person submits a request; a reviewer decides, and the person sees the decision.",
          "steps": [
            {"actor": "The person", "action": "describes what they need and submits the request", "result": "The reviewer can see the request"},
            {"actor": "The reviewer", "action": "makes a decision", "result": "The person sees the decision", "branches": [
              {"outcome": "Approved", "result": "The person sees confirmation"},
              {"outcome": "Declined", "result": "The person sees the reason and can revise the request"}
            ]}
          ],
          "visibility": "The person and the reviewer see the request; who else can see it is not established.",
          "consequences": "A decision takes reviewer time, and a declined request needs revision.",
          "evidence": "Authored pattern; no external research was performed.",
          "outcomes": ["The person completes a time-off request"]
        }
      ],
      "excluded": ["d666666666666666666666666"],
      "exclusions": [
        {"idea": "d666666666666666666666666", "description": "HR reviews every request", "consequence": "Requests wait on HR", "provenance": "human"}
      ],
      "unknowns": ["Who can see a request beyond the people named in the walkthrough is not established."]
    },
    "challenges": [
      {"idea": "d444444444444444444444444", "assumption": "Everyone can see each request", "consequence": "Dates are exposed", "recommendation": "Confirm whether the visibility holds, or explain the correction."}
    ],
    "settled": ["d555555555555555555555555"],
    "recommendation": {
      "producer": "soda-authored-v1",
      "intent": "d000000000000000000000000",
      "revision": "d111111111111111111111111",
      "objective": "Time-off requests my manager can approve",
      "recommended": "shared_review",
      "alternative": {
        "key": "shared_review",
        "name": "Shared review workflow",
        "audience": "A person making a request and another person deciding on it.",
        "summary": "The person submits a request; a reviewer decides, and the person sees the decision.",
        "steps": [
          {"actor": "The person", "action": "describes what they need and submits the request", "result": "The reviewer can see the request"},
          {"actor": "The reviewer", "action": "makes a decision", "result": "The person sees the decision", "branches": [
            {"outcome": "Approved", "result": "The person sees confirmation"},
            {"outcome": "Declined", "result": "The person sees the reason and can revise the request"}
          ]}
        ],
        "visibility": "The person and the reviewer see the request; who else can see it is not established.",
        "consequences": "A decision takes reviewer time, and a declined request needs revision.",
        "evidence": "Authored pattern; no external research was performed.",
        "outcomes": ["The person completes a time-off request"]
      },
      "outcomes": ["The person completes a time-off request"],
      "constraints": [],
      "excluded": ["d666666666666666666666666"],
      "milestones": [
        {"name": "The person: describes what they need and submits the request", "result": "The reviewer can see the request"},
        {"name": "The reviewer: makes a decision", "result": "The person sees the decision"},
        {"name": "If Approved", "result": "The person sees confirmation"},
        {"name": "If Declined", "result": "The person sees the reason and can revise the request"}
      ],
      "advice": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
    },
    "consistency": {
      "producer": "soda-authored-v1",
      "intent": "d000000000000000000000000",
      "revision": "d111111111111111111111111",
      "mismatches": [],
      "confidence": 0.9,
      "advice": "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
    },
    "saved_unix": 1791824831
  }
}`

func decodeHFlow(t *testing.T, raw string) SavedReviewRead {
	t.Helper()
	var read SavedReviewRead
	if err := json.Unmarshal([]byte(raw), &read); err != nil {
		t.Fatal(err)
	}
	return read
}

func mutateHFlow(t *testing.T, mutate func(doc map[string]any)) string {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal([]byte(hflowCurrent), &doc); err != nil {
		t.Fatal(err)
	}
	mutate(doc)
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func reviewDoc(doc map[string]any) map[string]any {
	return doc["review"].(map[string]any)
}

func TestHFlowCurrentReadBindsBasis(t *testing.T) {
	read := decodeHFlow(t, hflowCurrent)
	if err := read.Validate(); err != nil {
		t.Fatal(err)
	}
	review := read.Review
	if review.BasisRevision == review.ResultRevision || read.HeadRevision != review.ResultRevision || read.Status != ReviewCurrent {
		t.Fatalf("current identity wrong: %+v", read)
	}
	if review.Content.Revision != review.BasisRevision || review.Recommendation.Revision != review.BasisRevision || review.Consistency.Revision != review.BasisRevision {
		t.Fatalf("payload not bound to basis: %+v", review)
	}
	if len(review.Content.Alternatives) != 2 || len(review.Recommendation.Milestones) != 4 || len(review.Challenges) != 1 || len(review.Settled) != 1 {
		t.Fatalf("payload shape wrong: %+v", review)
	}
}

func TestHFlowStaleHeadKeepsReview(t *testing.T) {
	raw := mutateHFlow(t, func(doc map[string]any) {
		doc["head_revision"] = "d333333333333333333333333"
		doc["status"] = "stale"
	})
	read := decodeHFlow(t, raw)
	if err := read.Validate(); err != nil {
		t.Fatal(err)
	}
	if read.Review == nil || read.Review.ResultRevision == read.HeadRevision {
		t.Fatalf("stale read lost the review: %+v", read)
	}
}

func TestHFlowMissingReadCarriesNoReview(t *testing.T) {
	raw := `{"intent": "d000000000000000000000000", "head_revision": "d333333333333333333333333", "status": "missing"}`
	read := decodeHFlow(t, raw)
	if err := read.Validate(); err != nil {
		t.Fatal(err)
	}
	if read.Review != nil {
		t.Fatalf("missing read carries a review: %+v", read)
	}
}

func TestHFlowRejectsSubstitutedIdentities(t *testing.T) {
	cases := map[string]string{
		"content bound to result": mutateHFlow(t, func(doc map[string]any) {
			reviewDoc(doc)["content"].(map[string]any)["revision"] = "d222222222222222222222222"
		}),
		"recommendation bound to result": mutateHFlow(t, func(doc map[string]any) {
			reviewDoc(doc)["recommendation"].(map[string]any)["revision"] = "d222222222222222222222222"
		}),
		"consistency bound to result": mutateHFlow(t, func(doc map[string]any) {
			reviewDoc(doc)["consistency"].(map[string]any)["revision"] = "d222222222222222222222222"
		}),
		"basis equals result": mutateHFlow(t, func(doc map[string]any) {
			reviewDoc(doc)["basis_revision"] = "d222222222222222222222222"
		}),
		"current claim past the head": mutateHFlow(t, func(doc map[string]any) {
			doc["head_revision"] = "d333333333333333333333333"
		}),
		"stale claim at the head": mutateHFlow(t, func(doc map[string]any) {
			doc["status"] = "stale"
		}),
		"missing carrying a review": mutateHFlow(t, func(doc map[string]any) {
			doc["status"] = "missing"
		}),
		"challenge colliding with settlement": mutateHFlow(t, func(doc map[string]any) {
			reviewDoc(doc)["settled"] = []any{"d444444444444444444444444"}
		}),
	}
	for name, raw := range cases {
		if err := decodeHFlow(t, raw).Validate(); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}

func TestHFlowRejectsBadAdviceProvenance(t *testing.T) {
	raw := mutateHFlow(t, func(doc map[string]any) {
		reviewDoc(doc)["recommendation"].(map[string]any)["advice"] = "not-a-digest"
	})
	if err := decodeHFlow(t, raw).Validate(); err == nil || !strings.Contains(err.Error(), "advice") {
		t.Fatalf("bad recommendation advice accepted: %v", err)
	}
	raw = mutateHFlow(t, func(doc map[string]any) {
		reviewDoc(doc)["consistency"].(map[string]any)["advice"] = "not-a-digest"
	})
	if err := decodeHFlow(t, raw).Validate(); err == nil || !strings.Contains(err.Error(), "advice") {
		t.Fatalf("bad consistency advice accepted: %v", err)
	}
}

func TestClassifyReview(t *testing.T) {
	if got := ClassifyReview("d222222222222222222222222", "d222222222222222222222222", true); got != ReviewCurrent {
		t.Fatalf("current=%s", got)
	}
	if got := ClassifyReview("d333333333333333333333333", "d222222222222222222222222", true); got != ReviewStale {
		t.Fatalf("stale=%s", got)
	}
	if got := ClassifyReview("d333333333333333333333333", "d222222222222222222222222", false); got != ReviewMissing {
		t.Fatalf("missing=%s", got)
	}
}
