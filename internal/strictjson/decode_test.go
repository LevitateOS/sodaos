package strictjson

import (
	"bytes"
	"errors"
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

func TestDecodeNestedDepthBoundary(t *testing.T) {
	input := func(levels int) string {
		return `{"meta":` + strings.Repeat(`{"k":`, levels) + `0` + strings.Repeat(`}`, levels) + `}`
	}
	var accepted map[string]any
	if err := Decode(strings.NewReader(input(100)), &accepted); err != nil {
		t.Fatalf("100 nested value levels refused: %v", err)
	}
	var tooDeep map[string]any
	if err := Decode(strings.NewReader(input(101)), &tooDeep); err == nil || !strings.Contains(err.Error(), "nested too deeply") {
		t.Fatalf("101 nested value levels were not refused: %v", err)
	}
}

func TestDecodePreservesNumberTokenAdmission(t *testing.T) {
	var request struct {
		ID string `json:"id"`
	}
	if err := Decode(strings.NewReader(`{"id":1e999}`), &request); err == nil {
		t.Fatal("out-of-range number accepted during duplicate scan")
	}
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

type brokenReader struct {
	data []byte
}

func (r *brokenReader) Read(p []byte) (int, error) {
	if len(r.data) != 0 {
		n := copy(p, r.data)
		r.data = r.data[n:]
		return n, nil
	}
	return 0, errors.New("fixture read failure")
}

func TestDecodePropagatesInputReadFailure(t *testing.T) {
	var request testRequest
	err := Decode(&brokenReader{data: []byte(`{"id":"one"}`)}, &request)
	require.ErrorContains(t, err, "read request")
	require.ErrorContains(t, err, "fixture read failure")
}
