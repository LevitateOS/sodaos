package strictjson

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type testRequest struct {
	ID string `json:"id"`
}

func TestDecodeAcceptsOneKnownObject(t *testing.T) {
	var request testRequest
	require.NoError(t, Decode(strings.NewReader(`{"id":"one"}`), &request))
	require.Equal(t, testRequest{ID: "one"}, request)
}

func TestDecodeRejectsInvalidRequestShapes(t *testing.T) {
	for name, input := range map[string]string{
		"duplicate field": `{"id":"one","id":"two"}`,
		"unknown field":   `{"id":"one","project_id":"site"}`,
		"array":           `[]`,
		"trailing object": `{"id":"one"}{"id":"two"}`,
		"unclosed object": `{"id":"one"`,
	} {
		t.Run(name, func(t *testing.T) {
			var request testRequest
			require.Error(t, Decode(strings.NewReader(input), &request))
		})
	}
}

type nestedRequest struct {
	ID   string            `json:"id"`
	Meta map[string]string `json:"meta"`
}

type listedRequest struct {
	ID    string              `json:"id"`
	Items []map[string]string `json:"items"`
}

func TestDecodeRejectsNestedDuplicateFields(t *testing.T) {
	for name, input := range map[string]string{
		"nested duplicate":        `{"id":"one","meta":{"key":"1","key":"2"}}`,
		"deeply nested duplicate": `{"id":"one","meta":{"outer":{"key":"1","key":"2"}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			var request nestedRequest
			require.ErrorContains(t, Decode(strings.NewReader(input), &request), "duplicate")
		})
	}
	t.Run("duplicate in nested list", func(t *testing.T) {
		var request listedRequest
		require.ErrorContains(t, Decode(strings.NewReader(`{"id":"one","items":[{"key":"1"},{"key":"1","key":"2"}]}`), &request), "duplicate")
	})
}

func TestDecodeAcceptsUniqueNestedFields(t *testing.T) {
	var request nestedRequest
	require.NoError(t, Decode(strings.NewReader(`{"id":"one","meta":{"first":"1","second":"2"}}`), &request))
	require.Equal(t, nestedRequest{ID: "one", Meta: map[string]string{"first": "1", "second": "2"}}, request)
}

func TestDecodeRejectsInvalidUTF8AndOversizedRequests(t *testing.T) {
	invalidUTF8 := append([]byte(`{"id":"`), 0xff)
	invalidUTF8 = append(invalidUTF8, []byte(`"}`)...)
	var request testRequest
	require.ErrorContains(t, Decode(bytes.NewReader(invalidUTF8), &request), "valid UTF-8")

	oversized := strings.NewReader(`{"id":"` + strings.Repeat("a", maximumRequestBytes) + `"}`)
	require.ErrorContains(t, Decode(oversized, &request), "exceeds 1 MiB")
}
