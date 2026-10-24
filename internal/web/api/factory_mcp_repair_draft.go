package api

import (
	"encoding/json"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/levitateos/sodaos/internal/factory"
)

func repairDraftPayloadSchema() *jsonschema.Schema {
	shape := repairPreviewPayloadSchema()
	shape.Properties["diagnosis_refs"] = &jsonschema.Schema{Type: "array", MaxItems: ptrInt(16), Items: &jsonschema.Schema{Type: "string", Pattern: "^diagnosis:[a-f0-9]{32}$"}, Description: "Optional exact diagnosis references recorded by this assignment under the same cause/input. A reference grants no repair authority."}
	return shape
}

func checkRepairDraftPayload(payload map[string]any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = factory.DecodeRepairDraft(raw)
	return err
}
