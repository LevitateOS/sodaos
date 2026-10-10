# soda-avatars

Generate a local inspection catalog of Soda's robot avatars using the production
Go renderer. Designers and developers can inspect component variants, palette
combinations and deterministic examples at several display sizes.

## Generate a catalog

Run from the repository root with the Go toolchain selected by
[go.mod](../../go.mod):

```sh
go run ./tools/soda-avatars
```

The command takes no options. It creates a fresh `preview-*` directory below
`.artifacts/avatars/`, writes SVG files and an `index.html`, then prints the
absolute path to that HTML file. An error prints to stderr and exits `1`.
Arguments are ignored; there is no `--out` or `--help` parser.

Open the printed HTML file in a browser. Its controls switch between light and
dark surfaces and square or circular crops. Select an avatar to download its SVG.
The catalog includes:

- Every head, face, eyes, mouth, earcaps and accessory variant.
- All 32 background/accent combinations.
- 100 stable robots displayed at 128, 64, 32 and 24 pixels.

Each invocation preserves earlier previews. The tool writes only the local
catalog; it does not start a server or update Forgejo profiles. Catalog samples
use synthetic seeds, while product avatars derive their seed from the admitted
Forgejo email hash.

The [avatar package guide](../../internal/avatar/README.md) owns artwork
provenance and editable style guidance. The
[avatar behavior guide](../../docs/design/avatars.md) describes the public image
route, versioning and Forgejo activation.
