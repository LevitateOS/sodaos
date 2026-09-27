package host

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNativeReaderStreamsInput(t *testing.T) {
	input := strings.Repeat("synthetic release\n", 10000)
	out, err := (Native{}).RunReader(context.Background(), strings.NewReader(input), "/bin/cat")
	if err != nil || string(out) != input {
		t.Fatal("reader input not delivered", err)
	}
}

func TestNativeReaderFailureAndCancellation(t *testing.T) {
	if _, err := (Native{}).RunReader(context.Background(), strings.NewReader("fixture"), "/bin/sh", "-c", "exit 3"); err == nil {
		t.Fatal("consumer failure accepted")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := (Native{}).RunReader(ctx, strings.NewReader("fixture"), "/bin/sleep", "30")
	if err == nil || !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatal("native process cancellation not observed")
	}
	if time.Since(started) > 2*time.Second {
		t.Fatal("native consumer survived cancellation")
	}
}
