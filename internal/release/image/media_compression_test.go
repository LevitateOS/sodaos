package image

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMediaCompressionAdmissionAndMetadata(t *testing.T) {
	valid := Request{Development: true, Target: "media", RootfsBaseURL: "https://example.invalid", MediaCompression: "fast"}
	require.NoError(t, valid.ValidateTarget())
	for _, r := range []Request{
		{MediaCompression: "fast", RootfsBaseURL: valid.RootfsBaseURL},
		{Development: true, Target: "candidate", MediaCompression: "fast"},
		{Development: true, Target: "media", RootfsBaseURL: valid.RootfsBaseURL, MediaCompression: "unknown"},
	} {
		require.Error(t, r.ValidateTarget())
	}
	original := map[string]json.RawMessage{"live-rootfs-fstype": json.RawMessage(`"erofs"`), "live-rootfs-fsoptions": json.RawMessage(`"` + defaultRootfsOptions + `"`), "other": json.RawMessage(`{"preserve":true}`)}
	before, err := json.Marshal(original)
	require.NoError(t, err)
	require.NoError(t, setMediaCompression(original, ""))
	after, err := json.Marshal(original)
	require.NoError(t, err)
	require.Equal(t, before, after)
	require.NoError(t, setMediaCompression(original, "fast"))
	want := json.RawMessage(`"` + fastRootfsOptions + `"`)
	require.Equal(t, want, original["live-rootfs-fsoptions"])
	require.Equal(t, json.RawMessage(`{"preserve":true}`), original["other"])
	// Refuse an unreviewed upstream default rather than silently changing it.
	require.Error(t, setMediaCompression(original, "fast"))
	require.Equal(t, want, original["live-rootfs-fsoptions"])
}

func TestImageConfigNativeReadback(t *testing.T) {
	context, out := t.TempDir(), t.TempDir()
	raw := `{"live-rootfs-fstype":"erofs","live-rootfs-fsoptions":"` + fastRootfsOptions + `","other":true}`
	require.NoError(t, ownedWrite(filepath.Join(context, "rootfs", imageConfigPath), []byte(raw+"\n"), 0o644))
	require.ErrorContains(t, recordImageConfig(context, out, "{}"), "differs")
	require.NoFileExists(t, filepath.Join(out, "image-config.json"))
	require.NoError(t, recordImageConfig(context, out, raw))
	fs, options, err := rootfsSettings(out)
	require.NoError(t, err)
	require.Equal(t, "erofs", fs)
	require.Equal(t, fastRootfsOptions, options)
	require.Error(t, recordImageConfig(context, out, raw)) // no replacing receipts
}

func TestMediaEventsObserveBoundedPublicWindows(t *testing.T) {
	var log, events bytes.Buffer
	w := &mediaEventWriter{log: &log, events: &events, start: time.Now()}
	text := "private text is not an event\nGenerating osmet file for synthetic.raw image\nPacking successful!\nCreating erofs with options\nSubstituting ISO kernel arguments: private text\ngenisoimage 1.1.11\n100 extents written (1 MB)\n" + strings.Repeat("x", 9000) + "\n"
	for i := 0; i < len(text); i += 13 {
		_, err := w.Write([]byte(text[i:min(i+13, len(text))]))
		require.NoError(t, err)
	}
	require.Equal(t, text, log.String())
	require.NotContains(t, events.String(), "private")
	var names []string
	decoder := json.NewDecoder(&events)
	for decoder.More() {
		var event struct {
			Event   string
			Seconds float64
		}
		require.NoError(t, decoder.Decode(&event))
		require.GreaterOrEqual(t, event.Seconds, 0.0)
		names = append(names, event.Event)
	}
	require.Equal(t, []string{"osmet-start", "osmet-end", "rootfs-start", "rootfs-end", "iso-start", "iso-end"}, names)
	f, err := os.CreateTemp(t.TempDir(), "closed")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	w.events = f
	_, err = w.Write([]byte("Packing successful!\n"))
	require.True(t, errors.Is(err, os.ErrClosed))
}
