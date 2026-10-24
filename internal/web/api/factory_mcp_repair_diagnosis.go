package api

import (
	"encoding/json"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/levitateos/sodaos/internal/factory"
)

func repairDiagnosisPayloadSchema() *jsonschema.Schema {
	schema := mcpCarrier[factory.RepairDiagnosis]()
	schema.Properties["state"].Enum = []any{"suspected", "supported", "contradicted", "incomplete"}
	return schema
}

func checkRepairDiagnosisPayload(payload map[string]any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = factory.DecodeRepairDiagnosis(raw)
	return err
}
