package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"
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
	Version  string `json:"version"`
	Scope    string `json:"scope"`
	Stream   string `json:"stream"`
	Snapshot string `json:"snapshot"`
	Offset   int    `json:"offset"`
	Expires  int64  `json:"expires"`
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
	return &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{"cursor": {Type: "string", MaxLength: ptrInt(1024), Description: "Authenticated continuation from current_role_page.next_cursor. Bound to your exact scope/input snapshot and expires within five minutes; stale pages require a fresh context read."}}, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}}
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

func contextCursorMAC(key, payload []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte("soda-factory-required-context-v1\x00"))
	_, _ = mac.Write(payload)
	return mac.Sum(nil)
}

func pageCurrentRole(cap factory.MCPCapability, role factory.CurrentRoleContext, args mcpCallArgs, key []byte, now time.Time) (currentContextPage, error) {
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
	if cursor != "" && len(key) < 32 {
		return page, errors.New("context cursor signing is unavailable")
	}
	offset := 0
	expires := min(cap.DeadlineUnix, now.Add(5*time.Minute).Unix())
	if cursor != "" {
		payload, signature, ok := strings.Cut(cursor, ".")
		raw, decodeErr := base64.RawURLEncoding.DecodeString(payload)
		mac, macErr := base64.RawURLEncoding.DecodeString(signature)
		var position currentContextCursor
		if !ok || decodeErr != nil || macErr != nil || !hmac.Equal(mac, contextCursorMAC(key, raw)) || json.Unmarshal(raw, &position) != nil || position.Version != page.Version || position.Scope != cap.ScopeDigest() || position.Stream != page.Stream || position.Snapshot != snapshot || position.Offset < 0 || position.Offset >= len(records) || position.Expires > cap.DeadlineUnix || now.Unix() >= position.Expires {
			return page, errors.New("context cursor is stale or unavailable")
		}
		offset, expires = position.Offset, position.Expires
	}
	end := contextPageEnd(records, offset)
	page.Records = records[offset:end]
	page.PageComplete = true
	if end < len(records) {
		if len(key) < 32 {
			return page, errors.New("context cursor signing is unavailable")
		}
		payload, _ := json.Marshal(currentContextCursor{Version: page.Version, Scope: cap.ScopeDigest(), Stream: page.Stream, Snapshot: snapshot, Offset: end, Expires: expires})
		page.NextCursor = base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(contextCursorMAC(key, payload))
	} else {
		page.CollectionComplete = true
	}
	return page, nil
}
