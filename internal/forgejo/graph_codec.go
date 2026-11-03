package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
)

// GraphCodec owns Fountain's existing graph packet and receipt encoding. It
// has no transport, credentials or admission; IssueGraphWriter owns those.
type GraphCodec struct{}

var _ factory.GraphCodec = GraphCodec{}

func (GraphCodec) EncodeGraph(plan collaboration.GraphIntent, repository int64) (json.RawMessage, error) {
	if err := plan.ValidateShape(collaboration.RepositoryID(repository)); err != nil {
		return nil, err
	}
	var packet sdk.IssueGraphPayload
	for _, create := range plan.Creates {
		packet.Creates = append(packet.Creates, sdk.IssueGraphCreate{Key: create.Key, Title: create.Title, Body: create.Body})
	}
	for _, edit := range plan.Edits {
		id, err := originalCreationID(edit.Issue.ExternalID)
		if err != nil {
			return nil, err
		}
		version, err := strconv.Atoi(string(edit.Expected.Revision))
		if err != nil || version < 0 || strconv.Itoa(version) != string(edit.Expected.Revision) {
			return nil, ErrInvalidResponse
		}
		packet.Edits = append(packet.Edits, sdk.IssueGraphEdit{IssueID: id, ExpectedTitle: edit.Expected.Title, ExpectedBody: edit.Expected.Body, ExpectedContentVersion: version, Title: edit.Title, Body: edit.Body})
	}
	ref := func(value collaboration.GraphRef) (sdk.IssueGraphRef, error) {
		if value.Issue == nil {
			return sdk.IssueGraphRef{Key: value.Key}, nil
		}
		id, err := originalCreationID(value.Issue.ExternalID)
		return sdk.IssueGraphRef{IssueID: id}, err
	}
	for _, edge := range plan.Dependencies {
		issue, err := ref(edge.Issue)
		if err != nil {
			return nil, err
		}
		prerequisite, err := ref(edge.Prerequisite)
		if err != nil {
			return nil, err
		}
		packet.Dependencies = append(packet.Dependencies, sdk.IssueGraphDependency{Issue: issue, Prerequisite: prerequisite, Remove: edge.Remove})
	}
	if err := sdk.ValidateIssueGraphPayload(packet); err != nil {
		return nil, err
	}
	return json.Marshal(packet)
}

func (GraphCodec) InspectGraph(raw json.RawMessage) (factory.GraphPacketFacts, error) {
	var facts factory.GraphPacketFacts
	var packet sdk.IssueGraphPayload
	if decodeGraphJSON(raw, &packet) != nil {
		return facts, ErrInvalidResponse
	}
	// The empty packet is retained only by an explicitly local-only journal
	// step. Its {} digest predates this seam; it must never become a send.
	empty := len(packet.Creates)+len(packet.Edits)+len(packet.Dependencies) == 0 && packet.Intake == nil
	if !empty && sdk.ValidateIssueGraphPayload(packet) != nil {
		return facts, ErrInvalidResponse
	}
	canonical, err := json.Marshal(packet)
	if err != nil {
		return facts, err
	}
	facts.Canonical, facts.DependencyChanges = canonical, len(packet.Dependencies)
	for _, create := range packet.Creates {
		facts.Creates = append(facts.Creates, collaboration.GraphCreate{Key: create.Key, Title: create.Title, Body: create.Body})
	}
	facts.Edits = make(map[int64]factory.GraphIssueText, len(packet.Edits))
	for _, edit := range packet.Edits {
		facts.Edits[edit.IssueID] = factory.GraphIssueText{Title: edit.Title, Body: edit.Body}
	}
	return facts, nil
}

func (GraphCodec) DecodeGraphReceipt(_ context.Context, raw json.RawMessage) (factory.GraphReceiptFacts, error) {
	var receipt sdk.IssueGraphReceipt
	var facts factory.GraphReceiptFacts
	if decodeGraphJSON(raw, &receipt) != nil || receipt.RepositoryID <= 0 || receipt.ActorID <= 0 {
		return facts, ErrInvalidResponse
	}
	facts.Original, facts.Repository, facts.Actor = bytes.Clone(raw), receipt.RepositoryID, receipt.ActorID
	facts.Creates = map[string]factory.GeneratedIssueIdentity{}
	seen := map[int64]bool{}
	for _, entry := range receipt.Issues {
		if entry.IssueID <= 0 || entry.IssueIndex <= 0 || entry.ContentVersion < 0 || seen[entry.IssueID] {
			return factory.GraphReceiptFacts{}, ErrInvalidResponse
		}
		seen[entry.IssueID] = true
		identity := factory.GeneratedIssueIdentity{IssueID: entry.IssueID, IssueIndex: entry.IssueIndex, ContentVersion: entry.ContentVersion}
		if entry.Key == "" {
			facts.Edits = append(facts.Edits, identity)
		} else {
			if _, exists := facts.Creates[entry.Key]; exists {
				return factory.GraphReceiptFacts{}, ErrInvalidResponse
			}
			facts.Creates[entry.Key] = identity
		}
	}
	return facts, nil
}

func decodeGraphJSON(raw json.RawMessage, value any) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return ErrInvalidResponse
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return ErrInvalidResponse
	}
	return nil
}
