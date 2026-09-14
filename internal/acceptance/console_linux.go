package acceptance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	_ "image/png"
	"os"
	"time"
	"unicode"
)

// ConsoleRegion identifies a reviewed prompt in the fixed qualification display.
// Matching happens before input, particularly before sending a private password.
// This is not OCR or an alternate installer state machine: the installer still
// owns validation, transitions and disk writes. A changed display fails closed.
type ConsoleRegion struct {
	X, Y, Width, Height int
	SHA256              string
}

func ConsoleRegionHash(frame image.Image, r ConsoleRegion) (string, error) {
	rect := image.Rect(r.X, r.Y, r.X+r.Width, r.Y+r.Height)
	if r.Width <= 0 || r.Height <= 0 || !rect.In(frame.Bounds()) {
		return "", errors.New("console region outside display")
	}
	h := sha256.New()
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			red, green, blue, _ := frame.At(x, y).RGBA()
			_, _ = h.Write([]byte{byte(red >> 8), byte(green >> 8), byte(blue >> 8)})
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// WaitConsole uses QEMU's native screenshot output. path is a disposable private
// scratch file belonging to this VM, not published evidence or a prior run's file.
func (q QMPClient) WaitConsole(ctx context.Context, path string, r ConsoleRegion) error {
	for {
		if err := q.Execute(ctx, "screendump", "console-prompt", map[string]any{"filename": path, "format": "png"}, nil); err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		frame, _, err := image.Decode(f)
		closeErr := f.Close()
		if err != nil || closeErr != nil {
			return errors.Join(err, closeErr)
		}
		digest, err := ConsoleRegionHash(frame, r)
		if err != nil {
			return err
		}
		if digest == r.SHA256 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
}

// TypeConsole sends private input only through the owned QMP socket. Neither the
// input nor individual key events may be logged as evidence. Call WaitConsole
// first when terminal echo or the active prompt matters.
func (q QMPClient) TypeConsole(ctx context.Context, text string) error {
	for _, ch := range text {
		key := string(ch)
		shift := unicode.IsUpper(ch)
		if shift {
			key = string(unicode.ToLower(ch))
		}
		plain := map[rune]string{'\n': "ret", ' ': "spc", '-': "minus", '.': "dot", '/': "slash", '=': "equal", ',': "comma", ';': "semicolon", '\'': "apostrophe", '[': "bracket_left", ']': "bracket_right", '\\': "backslash"}
		shifted := map[rune]string{'_': "minus", ':': "semicolon", '"': "apostrophe", '>': "dot", '<': "comma", '|': "backslash", '(': "9", ')': "0", '*': "8", '$': "4", '&': "7", '!': "1", '?': "slash", '{': "bracket_left", '}': "bracket_right", '+': "equal", '#': "3", '@': "2", '%': "5", '^': "6", '~': "grave"}
		if v, ok := plain[ch]; ok {
			key = v
		} else if v, ok := shifted[ch]; ok {
			key = v
			shift = true
		} else if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9') {
			return errors.New("unsupported console character")
		}
		keys := []map[string]string{}
		if shift {
			keys = append(keys, map[string]string{"type": "qcode", "data": "shift"})
		}
		keys = append(keys, map[string]string{"type": "qcode", "data": key})
		if err := q.Execute(ctx, "send-key", "console-input", map[string]any{"keys": keys, "hold-time": 10}, nil); err != nil {
			return errors.New("console input failed")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(25 * time.Millisecond):
		}
	}
	return nil
}
