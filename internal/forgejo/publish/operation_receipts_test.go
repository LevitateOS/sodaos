package publish

import (
	"encoding/json"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestDecodePublishReceiptRefusesLookalikes(t *testing.T) {
	receipt := PublishReceipt{
		Ref: "refs/heads/soda/factory/a", OldOID: strings.Repeat("0", 40),
		NewOID: strings.Repeat("2", 40), ComparisonRef: "refs/heads/main",
		ComparisonOID: strings.Repeat("1", 40), ActorID: 5, RepositoryID: 7,
	}
	raw, _ := json.Marshal(receipt)
	outcome := factory.OperationOutcome{Effect: factory.OpEffectCommitted, Receipt: raw}
	branch, err := DecodePublishReceipt(outcome, receipt.Ref, extensions.PublishExpectedOldAbsent, receipt.NewOID, receipt.ComparisonRef, receipt.ComparisonOID, 5, 7)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if branch.NewOID != receipt.NewOID || branch.Comparison != receipt.ComparisonOID {
		t.Fatalf("branch: %+v", branch)
	}
	for name, mutate := range map[string]func(*PublishReceipt){
		"ref":            func(r *PublishReceipt) { r.Ref = "refs/heads/other" },
		"new":            func(r *PublishReceipt) { r.NewOID = strings.Repeat("3", 40) },
		"comparison":     func(r *PublishReceipt) { r.ComparisonRef = "refs/heads/other" },
		"comparison_oid": func(r *PublishReceipt) { r.ComparisonOID = strings.Repeat("3", 40) },
		"actor":          func(r *PublishReceipt) { r.ActorID = 6 },
		"repo":           func(r *PublishReceipt) { r.RepositoryID = 8 },
		"old":            func(r *PublishReceipt) { r.OldOID = strings.Repeat("4", 40) },
	} {
		mutated := receipt
		mutate(&mutated)
		raw, _ := json.Marshal(mutated)
		bad := factory.OperationOutcome{Effect: factory.OpEffectCommitted, Receipt: raw}
		if _, err := DecodePublishReceipt(bad, receipt.Ref, extensions.PublishExpectedOldAbsent, receipt.NewOID, receipt.ComparisonRef, receipt.ComparisonOID, 5, 7); err == nil {
			t.Errorf("case %s adopted", name)
		}
	}
	pending := factory.OperationOutcome{Effect: factory.OpEffectPending, Receipt: raw}
	if _, err := DecodePublishReceipt(pending, receipt.Ref, extensions.PublishExpectedOldAbsent, receipt.NewOID, receipt.ComparisonRef, receipt.ComparisonOID, 5, 7); err == nil {
		t.Fatal("pending outcome decoded")
	}
	missing := factory.OperationOutcome{Effect: factory.OpEffectCommitted}
	if _, err := DecodePublishReceipt(missing, receipt.Ref, extensions.PublishExpectedOldAbsent, receipt.NewOID, receipt.ComparisonRef, receipt.ComparisonOID, 5, 7); err == nil {
		t.Fatal("missing receipt decoded")
	}
}

func TestDecodePRCreateReceiptRefusesLookalikes(t *testing.T) {
	receipt := PRCreateReceipt{
		HeadRef: "refs/heads/soda/factory/a", BaseRef: "refs/heads/main",
		HeadOID: strings.Repeat("2", 40), BaseOID: strings.Repeat("1", 40),
		AuthorID: 5, RepositoryID: 7, PRID: 8, IssueID: 10, PRNumber: 9,
	}
	raw, _ := json.Marshal(receipt)
	outcome := factory.OperationOutcome{Effect: factory.OpEffectCommitted, Receipt: raw}
	created, err := DecodePRCreateReceipt(outcome, receipt.HeadRef, receipt.BaseRef, receipt.HeadOID, receipt.BaseOID, 5, 7)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.PRNumber != 9 || created.PRID != 8 || created.IssueID != 10 {
		t.Fatalf("created: %+v", created)
	}
	for name, mutate := range map[string]func(*PRCreateReceipt){
		"head":   func(r *PRCreateReceipt) { r.HeadRef = "refs/heads/other" },
		"base":   func(r *PRCreateReceipt) { r.BaseOID = strings.Repeat("3", 40) },
		"repo":   func(r *PRCreateReceipt) { r.RepositoryID = 8 },
		"author": func(r *PRCreateReceipt) { r.AuthorID = 6 },
		"number": func(r *PRCreateReceipt) { r.PRNumber = 0 },
	} {
		mutated := receipt
		mutate(&mutated)
		raw, _ := json.Marshal(mutated)
		bad := factory.OperationOutcome{Effect: factory.OpEffectCommitted, Receipt: raw}
		if _, err := DecodePRCreateReceipt(bad, receipt.HeadRef, receipt.BaseRef, receipt.HeadOID, receipt.BaseOID, 5, 7); err == nil {
			t.Errorf("case %s adopted", name)
		}
	}
}

func TestReceiptRejectsTrailingDocument(t *testing.T) {
	branch := PublishReceipt{Ref: "refs/heads/soda/factory/a", OldOID: strings.Repeat("0", 40), NewOID: strings.Repeat("2", 40), ComparisonRef: "refs/heads/main", ComparisonOID: strings.Repeat("1", 40), ActorID: 5, RepositoryID: 7}
	raw, _ := json.Marshal(branch)
	outcome := factory.OperationOutcome{Effect: factory.OpEffectCommitted, Receipt: append(raw, []byte(" {}")...)}
	if _, err := DecodePublishReceipt(outcome, branch.Ref, "absent", branch.NewOID, branch.ComparisonRef, branch.ComparisonOID, 5, 7); err == nil {
		t.Fatal("trailing branch document adopted")
	}
	pr := PRCreateReceipt{HeadRef: branch.Ref, BaseRef: branch.ComparisonRef, HeadOID: branch.NewOID, BaseOID: branch.ComparisonOID, AuthorID: 5, RepositoryID: 7, PRID: 8, IssueID: 10, PRNumber: 9}
	raw, _ = json.Marshal(pr)
	outcome.Receipt = append(raw, []byte(" {}")...)
	if _, err := DecodePRCreateReceipt(outcome, pr.HeadRef, pr.BaseRef, pr.HeadOID, pr.BaseOID, 5, 7); err == nil {
		t.Fatal("trailing PR document adopted")
	}
}
