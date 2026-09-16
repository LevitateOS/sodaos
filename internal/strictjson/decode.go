// Package strictjson decodes bounded, single-object JSON request bodies.
package strictjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

const maximumRequestBytes = 1 << 20

// Decode accepts exactly one small JSON object, rejects duplicate and unknown
// fields, and never logs its contents.
func Decode(reader io.Reader, destination any) error {
	contents, err := io.ReadAll(io.LimitReader(reader, maximumRequestBytes+1))
	if err != nil {
		return fmt.Errorf("read request: %w", err)
	}
	if len(contents) > maximumRequestBytes {
		return errors.New("request exceeds 1 MiB")
	}
	if !utf8.Valid(contents) {
		return errors.New("request must contain valid UTF-8")
	}
	object, err := decodeUniqueObject(contents)
	if err != nil {
		return err
	}
	normalized, err := json.Marshal(object)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(normalized))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode request: %w", err)
	}
	return nil
}

func decodeUniqueObject(contents []byte) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(contents))
	if err := requireObject(decoder); err != nil {
		return nil, err
	}
	object, err := decodeFields(decoder)
	if err != nil {
		return nil, err
	}
	if err = finishObject(decoder); err != nil {
		return nil, err
	}
	return object, nil
}

func requireObject(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("decode request: %w", err)
	}
	delimiter, ok := token.(json.Delim)
	if !ok || delimiter != '{' {
		return errors.New("request must be one JSON object")
	}
	return nil
}

func decodeFields(decoder *json.Decoder) (map[string]json.RawMessage, error) {
	object := map[string]json.RawMessage{}
	for decoder.More() {
		field, err := decodeFieldName(decoder, object)
		if err != nil {
			return nil, err
		}
		var value json.RawMessage
		if err = decoder.Decode(&value); err != nil {
			return nil, fmt.Errorf("decode request field %q: %w", field, err)
		}
		if err = rejectDuplicateKeys(field, value, 0); err != nil {
			return nil, err
		}
		object[field] = value
	}
	return object, nil
}

func decodeFieldName(decoder *json.Decoder, object map[string]json.RawMessage) (string, error) {
	token, err := decoder.Token()
	if err != nil {
		return "", fmt.Errorf("decode request: %w", err)
	}
	field, valid := token.(string)
	if !valid {
		return "", errors.New("request field name must be a string")
	}
	if _, duplicate := object[field]; duplicate {
		return "", fmt.Errorf("duplicate request field %q", field)
	}
	return field, nil
}

// rejectDuplicateKeys extends the top-level duplicate ban to every nested
// object and array element: encoding/json keeps the last of any nested
// duplicates, so values must be scanned before they are accepted. Depth is
// capped because request bodies are small API objects, never deep documents.
func rejectDuplicateKeys(field string, raw json.RawMessage, depth int) error {
	if depth > 100 {
		return fmt.Errorf("decode request field %q: request is nested too deeply", field)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("decode request field %q: %w", field, err)
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		if err = rejectObjectDuplicates(decoder, field, depth); err != nil {
			return err
		}
	case '[':
		for decoder.More() {
			var element json.RawMessage
			if err = decoder.Decode(&element); err != nil {
				return fmt.Errorf("decode request field %q: %w", field, err)
			}
			if err = rejectDuplicateKeys(field, element, depth+1); err != nil {
				return err
			}
		}
		if _, err = decoder.Token(); err != nil {
			return fmt.Errorf("decode request field %q: %w", field, err)
		}
	}
	return nil
}

func rejectObjectDuplicates(decoder *json.Decoder, field string, depth int) error {
	seen := map[string]bool{}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("decode request field %q: %w", field, err)
		}
		name, valid := token.(string)
		if !valid {
			return fmt.Errorf("decode request field %q: request field name must be a string", field)
		}
		if seen[name] {
			return fmt.Errorf("duplicate request field %q", name)
		}
		seen[name] = true
		var value json.RawMessage
		if err = decoder.Decode(&value); err != nil {
			return fmt.Errorf("decode request field %q: %w", field, err)
		}
		if err = rejectDuplicateKeys(field, value, depth+1); err != nil {
			return err
		}
	}
	if _, err := decoder.Token(); err != nil {
		return fmt.Errorf("decode request field %q: %w", field, err)
	}
	return nil
}

func finishObject(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("decode request: %w", err)
	}
	delimiter, ok := token.(json.Delim)
	if !ok || delimiter != '}' {
		return errors.New("request object is not closed")
	}
	if _, err = decoder.Token(); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode request: %w", err)
	}
	return errors.New("request must contain exactly one JSON object")
}
