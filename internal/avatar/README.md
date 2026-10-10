# Soda robot v1

`style-v1.json` is the single editable source for the original robot paths,
component choices, colors and layer order. It uses DiceBear definition schema
1.5.1 with the Go core pinned in the root module. No Bottts paths or style
definition were copied. Original artwork and adapter code are Apache-2.0;
canonical Soda branding and other inherited assets retain their existing terms.

The visual reference is the nine-robot concept sheet generated during the Soda
avatar design conversation on 2026-09-08. The paths here were independently
constructed from geometric shapes, not traced from Bottts or the raster image.
Cream shells, navy screens, mint eyes and outlined accents adapt that concept to
small native profile images. The palette takes values from Soda's shared palette.

All components use a 256×256 coordinate system. Earcaps sit behind the shell;
the face, eyes, mouth and forehead accessory follow it. Keep features inside the
common face area and the whole head inside a circular crop. A pale keyline keeps
earcaps visible on dark backgrounds. Empty `plain` is an intentional accessory.

Generate an inspection catalog with the exact production renderer:

```sh
go run ./tools/soda-avatars
```

The tool creates a new directory below ignored `.artifacts/avatars/`, never
overwrites an earlier preview, and prints the path to its standalone HTML sheet.
It includes every part, all 32 background/accent combinations and 100 deterministic
avatars at four sizes, on light/dark surfaces with square/circular crops. SVG
links download the generated images. It is not a shipped application or server.

Before releasing v1, inspect the actual output and freeze the golden seed records.
After release, changes to component names, variants, colors or rendered bytes
require an explicit version decision; do not regenerate snapshots merely to make
an unexpected dependency change pass. Keep `idRandomization` disabled. The seed is
`soda-robot-v1:<lowercase Forgejo email hash>`: an avatar-email change can change
the result. Different hashes are not a guarantee of visually distinct robots.

## Use from a Soda Go caller

The package renders SVG without identity lookups or network access. Import
`github.com/levitateos/sodaos/internal/avatar` from inside this module:

```go
import "github.com/levitateos/sodaos/internal/avatar"

svg, err := avatar.Render("0123456789abcdef0123456789abcdef", 128)
```

Check `err` before serving or saving `svg`. The example seed is synthetic;
normal callers use Forgejo's 32-character hexadecimal email hash, never an
email address or username. `NormalizeHash` validates and lowercases that hash.
`Render` accepts sizes from 1 through 1024 and returns the same SVG for a given
hash/size across restarts. Rendering is safe for concurrent callers.

`Validate` checks the embedded artwork at service startup; `Definition` returns
the canonical embedded style as an immutable string. Ordinary users receive
avatars through the versioned route described by the
[API reference](../../docs/reference/api.md). The
[avatar design guide](../../docs/design/avatars.md) owns artwork/version rules,
and [branding](../../docs/design/branding.md) owns the shared visual assets.
