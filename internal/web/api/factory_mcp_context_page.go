package api

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"unicode/utf8"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/levitateos/sodaos/internal/factory"
)

const (
	maxContextCollectionBytes = 1024 * 1024
	maxContextRecords         = 512
	maxContextPages           = 64
)

var errCurrentContextBound = errors.New("required current collection exceeds its finite bound")

type currentContextCursor struct {
	Snapshot string `json:"snapshot"`
	Offset   int    `json:"offset"`
}

type currentContextRecord struct {
	Kind     string `json:"kind"`
	Field    string `json:"field"`
	Part     int    `json:"part"`
	Parts    int    `json:"parts"`
	Encoding string `json:"encoding"`
	Content  string `json:"content"`
}

type currentContextPage struct {
	Version            string                 `json:"version"`
	Stream             string                 `json:"stream"`
	Snapshot           string                 `json:"snapshot"`
	Records            []currentContextRecord `json:"records"`
	PageComplete       bool                   `json:"page_complete"`
	CollectionComplete bool                   `json:"collection_complete"`
	Availability       string                 `json:"availability"`
	MissingReason      string                 `json:"missing_reason,omitempty"`
	NextCursor         string                 `json:"next_cursor,omitempty"`
}

func contextPayloadSchema() *jsonschema.Schema {
	return &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{"cursor": {Type: "string", MaxLength: ptrInt(1024), Description: "Continuation from current_role_page.next_cursor for the same input snapshot. If the context changes, begin a fresh context read."}}, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}}
}

// Large typed fields use UTF-8-safe JSON fragments: concatenate a field's
// ordered content parts, then decode JSON. No prompt or arbitrary ref is read.
func currentRoleRecords(role factory.CurrentRoleContext) ([]currentContextRecord, string, error) {
	raw, err := json.Marshal(role)
	if err != nil {
		return nil, "", err
	}
	if len(raw) > maxContextCollectionBytes {
		return nil, "", errCurrentContextBound
	}
	sum := sha256.Sum256(raw)
	snapshot := hex.EncodeToString(sum[:])
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(raw, &fields); err != nil {
		return nil, "", err
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var records []currentContextRecord
	for _, field := range keys {
		value := fields[field]
		parts := []string{}
		for len(value) > 0 {
			end := min(4096, len(value))
			for end < len(value) && !utf8.RuneStart(value[end]) {
				end--
			}
			parts = append(parts, string(value[:end]))
			value = value[end:]
		}
		for index, part := range parts {
			records = append(records, currentContextRecord{Kind: "role_input", Field: field, Part: index, Parts: len(parts), Encoding: "json-fragment", Content: part})
		}
	}
	encoded, err := json.Marshal(records)
	if err != nil {
		return nil, "", err
	}
	if len(records) > maxContextRecords || len(encoded) > maxContextCollectionBytes {
		return nil, "", errCurrentContextBound
	}
	return records, snapshot, nil
}

func contextPageEnd(records []currentContextRecord, offset int) int {
	start, bytes := offset, 0
	for offset < len(records) && offset-start < factory.MCPMaxPageRefs {
		encoded, _ := json.Marshal(records[offset])
		if bytes+len(encoded)+1 > factory.MCPMaxPageBytes-2048 {
			break
		}
		bytes += len(encoded) + 1
		offset++
	}
	return offset
}

func pageCurrentRole(role factory.CurrentRoleContext, args mcpCallArgs) (currentContextPage, error) {
	page := currentContextPage{Version: "context-page-v1", Stream: "required_current", Records: []currentContextRecord{}, Availability: "available"}
	records, snapshot, err := currentRoleRecords(role)
	if err != nil {
		return page, err
	}
	for offset, pages := 0, 0; offset < len(records); pages++ {
		next := contextPageEnd(records, offset)
		if next == offset || pages >= maxContextPages {
			return page, errCurrentContextBound
		}
		offset = next
	}
	page.Snapshot = snapshot
	cursor := ""
	if len(args.Payload) > 0 {
		if len(args.Payload) != 1 {
			return page, errors.New("context accepts only its continuation cursor")
		}
		var ok bool
		cursor, ok = args.Payload["cursor"].(string)
		if !ok || len(cursor) > 1024 {
			return page, errors.New("invalid context cursor")
		}
	}
	offset := 0
	if cursor != "" {
		raw, decodeErr := base64.RawURLEncoding.DecodeString(cursor)
		var position currentContextCursor
		if decodeErr != nil || json.Unmarshal(raw, &position) != nil || position.Snapshot != snapshot || position.Offset < 0 || position.Offset >= len(records) {
			return page, errors.New("context cursor is stale or unavailable")
		}
		offset = position.Offset
	}
	end := contextPageEnd(records, offset)
	page.Records = records[offset:end]
	page.PageComplete = true
	if end < len(records) {
		payload, _ := json.Marshal(currentContextCursor{Snapshot: snapshot, Offset: end})
		page.NextCursor = base64.RawURLEncoding.EncodeToString(payload)
	} else {
		page.CollectionComplete = true
	}
	return page, nil
}
