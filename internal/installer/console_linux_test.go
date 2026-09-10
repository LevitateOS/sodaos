package installer

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestPasswordTerminalEchoAndCancellation(t *testing.T) {
	for _, cancelInput := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelInput), func(t *testing.T) {
			master, err := unix.Open("/dev/ptmx", unix.O_RDWR|unix.O_NOCTTY|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer unix.Close(master)
			if err := unix.IoctlSetPointerInt(master, unix.TIOCSPTLCK, 0); err != nil {
				t.Fatal(err)
			}
			number, err := unix.IoctlGetInt(master, unix.TIOCGPTN)
			if err != nil {
				t.Fatal(err)
			}
			slave, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", number), os.O_RDWR|unix.O_NOCTTY, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer slave.Close()
			original, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TCGETS)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c := console{tty: slave, ctx: ctx}
			type result struct {
				value string
				err   error
			}
			done := make(chan result, 1)
			go func() { value, err := c.secret("Operator password"); done <- result{value, err} }()
			var transcript strings.Builder
			read := func() {
				var buf [2048]byte
				n, _ := unix.Read(master, buf[:])
				if n > 0 {
					transcript.Write(buf[:n])
				}
			}
			deadline := time.Now().Add(2 * time.Second)
			for !strings.Contains(transcript.String(), "Operator password: ") && time.Now().Before(deadline) {
				read()
				time.Sleep(time.Millisecond)
			}
			if !strings.Contains(transcript.String(), "Operator password: ") {
				t.Fatal("no prompt")
			}
			hidden, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TCGETS)
			if err != nil || hidden.Lflag&unix.ECHO != 0 {
				t.Fatal("password echo enabled")
			}
			if cancelInput {
				cancel()
			} else {
				if _, err := unix.Write(master, []byte("synthetic password only\n")); err != nil {
					t.Fatal(err)
				}
			}
			select {
			case got := <-done:
				if cancelInput && got.err == nil {
					t.Fatal("cancellation ignored")
				}
				if !cancelInput && (got.err != nil || got.value != "synthetic password only") {
					t.Fatal("password input failed")
				}
			case <-time.After(3 * time.Second):
				t.Fatal("password input stuck")
			}
			restored, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TCGETS)
			if err != nil || restored.Lflag != original.Lflag {
				t.Fatal("terminal echo not restored")
			}
			read()
			if strings.Contains(transcript.String(), "synthetic password only") {
				t.Fatal("password echoed to terminal")
			}
		})
	}
}
