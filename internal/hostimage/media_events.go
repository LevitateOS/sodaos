package hostimage

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

func (w *mediaEventWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, err := w.log.Write(data)
	if err != nil {
		return n, err
	}
	for _, b := range data[:n] {
		if b != '\n' {
			if len(w.line) == 8192 {
				w.line = w.line[:0]
				w.dropped = true
			}
			if !w.dropped {
				w.line = append(w.line, b)
			}
			continue
		}
		event := ""
		if !w.dropped {
			line := string(w.line)
			switch {
			case strings.HasPrefix(line, "Generating osmet file for "):
				event = "osmet-start"
			case line == "Packing successful!":
				event = "osmet-end"
			case strings.HasPrefix(line, "Creating erofs with "):
				event = "rootfs-start"
			case strings.HasPrefix(line, "Substituting ISO kernel arguments:"):
				event = "rootfs-end"
			case strings.HasPrefix(line, "genisoimage "):
				event = "iso-start"
			case strings.Contains(line, " extents written ("):
				event = "iso-end"
			}
		}
		w.line, w.dropped = w.line[:0], false
		if event != "" {
			err = json.NewEncoder(w.events).Encode(struct {
				Event   string
				Seconds float64
			}{event, time.Since(w.start).Seconds()})
			if err != nil {
				return n, err
			}
		}
	}
	return n, nil
}
