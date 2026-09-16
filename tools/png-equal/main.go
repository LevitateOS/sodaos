package main

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"os"
)

// Bounds keep a corrupt or hostile PNG from exhausting memory: captures are
// small screenshots, so anything larger is refused before pixel allocation.
const (
	maxPNGFileBytes = 64 << 20
	maxPNGDimension = 16384
	maxPNGPixels    = 32 << 20
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: png-equal LEFT RIGHT")
		os.Exit(2)
	}
	equal, err := equalPNG(os.Args[1], os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if !equal {
		os.Exit(1)
	}
}

func decodeBoundedPNG(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maxPNGFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxPNGFileBytes {
		return nil, fmt.Errorf("PNG %s exceeds %d bytes", path, maxPNGFileBytes)
	}
	config, err := png.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > maxPNGDimension || config.Height > maxPNGDimension {
		return nil, fmt.Errorf("PNG %s dimensions %dx%d outside the %d bound", path, config.Width, config.Height, maxPNGDimension)
	}
	if int64(config.Width)*int64(config.Height) > maxPNGPixels {
		return nil, fmt.Errorf("PNG %s pixel count exceeds the %d bound", path, maxPNGPixels)
	}
	return png.Decode(bytes.NewReader(raw))
}

func equalPNG(leftPath, rightPath string) (bool, error) {
	left, err := decodeBoundedPNG(leftPath)
	if err != nil {
		return false, err
	}

	right, err := decodeBoundedPNG(rightPath)
	if err != nil {
		return false, err
	}

	if left.Bounds() != right.Bounds() {
		return false, nil
	}
	return bytes.Equal(rgbaPixels(left), rgbaPixels(right)), nil
}

func rgbaPixels(source image.Image) []byte {
	decoded := image.NewRGBA(source.Bounds())
	draw.Draw(decoded, decoded.Bounds(), source, source.Bounds().Min, draw.Src)
	return decoded.Pix
}
