package image

import (
	"encoding/json"
	"io"
	"strings"
	"sync"
	"time"
)

// Observe existing upstream progress without modifying the packager. These are
// log-arrival windows, not CPU profiles: the rootfs window includes CPIO/hashing,
// and osmet includes its checksum/readback. Never persist arbitrary log text here.
type mediaEventWriter struct {
	mu          sync.Mutex
	log, events io.Writer
	start       time.Time
	line        []byte
	dropped     bool
}

func mediaLogEvent(line string) string {
	switch {
	case strings.HasPrefix(line, "Generating osmet file for "):
		return "osmet-start"
	case line == "Packing successful!":
		return "osmet-end"
	case strings.HasPrefix(line, "Creating erofs with "):
		return "rootfs-start"
	case strings.HasPrefix(line, "Substituting ISO kernel arguments:"):
		return "rootfs-end"
	case strings.HasPrefix(line, "genisoimage "):
		return "iso-start"
	case strings.Contains(line, " extents written ("):
		return "iso-end"
	default:
		return ""
	}
}

func (w *mediaEventWriter) appendLineByte(b byte) {
	if len(w.line) == 8192 {
		w.line = w.line[:0]
		w.dropped = true
	}
	if !w.dropped {
		w.line = append(w.line, b)
	}
}

func (w *mediaEventWriter) finishLine() string {
	event := ""
	if !w.dropped {
		event = mediaLogEvent(string(w.line))
	}
	w.line, w.dropped = w.line[:0], false
	return event
}

func (w *mediaEventWriter) consume(b byte) error {
	if b != '\n' {
		w.appendLineByte(b)
		return nil
	}
	event := w.finishLine()
	if event == "" {
		return nil
	}
	return json.NewEncoder(w.events).Encode(struct {
		Event   string
		Seconds float64
	}{event, time.Since(w.start).Seconds()})
}

func (w *mediaEventWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, err := w.log.Write(data)
	if err != nil {
		return n, err
	}
	for _, b := range data[:n] {
		if err = w.consume(b); err != nil {
			return n, err
		}
	}
	return n, nil
}
