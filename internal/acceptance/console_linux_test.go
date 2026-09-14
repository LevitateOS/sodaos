package acceptance

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConsolePromptRegion(t *testing.T) {
	frame := image.NewRGBA(image.Rect(0, 0, 16, 16))
	r := ConsoleRegion{X: 2, Y: 2, Width: 4, Height: 4}
	first, err := ConsoleRegionHash(frame, r)
	require.NoError(t, err)
	frame.Set(0, 0, color.White)
	same, err := ConsoleRegionHash(frame, r)
	require.NoError(t, err)
	require.Equal(t, first, same)
	frame.Set(3, 3, color.White)
	changed, err := ConsoleRegionHash(frame, r)
	require.NoError(t, err)
	require.NotEqual(t, first, changed)
	_, err = ConsoleRegionHash(frame, ConsoleRegion{Width: 17, Height: 16})
	require.Error(t, err)
}

func TestConsoleUnsupportedInputRefused(t *testing.T) {
	require.ErrorContains(t, (QMPClient{}).TypeConsole(t.Context(), "\x00"), "unsupported console character")
}
