package factory

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

const MaxFeedArtifactBytes = 8 * 1024 * 1024
const MaxFeedArtifactChunk = 32 * 1024

type FeedArtifact struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MediaType string `json:"media_type"`
	Href      string `json:"href"`
	Size      int    `json:"size"`
}
type ArtifactUpload struct {
	UploadID      string `json:"upload_id"`
	Name          string `json:"name"`
	MediaType     string `json:"media_type"`
	Offset        int    `json:"offset"`
	ContentBase64 string `json:"content_base64"`
	Final         bool   `json:"final"`
}

func DecodeArtifactUpload(raw []byte) (ArtifactUpload, []byte, error) {
	var u ArtifactUpload
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&u); err != nil {
		return u, nil, err
	}
	if !ValidID(u.UploadID) || strings.TrimSpace(u.Name) == "" || len(u.Name) > 160 || strings.ContainsAny(u.Name, "/\\\r\n\x00") || u.Offset < 0 || u.Offset > MaxFeedArtifactBytes {
		return u, nil, errors.New("invalid artifact identity, name or offset")
	}
	switch u.MediaType {
	case "image/png", "image/jpeg", "image/webp", "image/gif", "video/mp4", "video/webm", "audio/mpeg", "audio/ogg", "audio/wav", "text/plain", "text/markdown", "application/json", "application/pdf", "application/octet-stream":
	default:
		return u, nil, errors.New("unsupported artifact media type")
	}
	content, err := base64.StdEncoding.DecodeString(u.ContentBase64)
	if err != nil || len(content) == 0 || len(content) > MaxFeedArtifactChunk || u.Offset+len(content) > MaxFeedArtifactBytes {
		return u, nil, errors.New("artifact chunk must contain 1–32768 bytes; total at most 8 MiB")
	}
	return u, content, nil
}
