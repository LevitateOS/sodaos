package assets

import "embed"

// Files contains the canonical artwork and theme tokens used by the dashboard.
//
//go:embed branding/source/*.svg branding/theme/palette.css branding/web/*.png
var Files embed.FS
