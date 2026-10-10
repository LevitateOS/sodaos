# png-equal

Compare two PNG images by decoded RGBA pixels. This development tool is useful
for checking screenshot or branding output when PNG compression and metadata
may differ but the visible image should remain identical.

## Compare images

From the repository root, using the Go toolchain selected in
[go.mod](../../go.mod):

```sh
go run ./tools/png-equal .artifacts/screenshots/before.png .artifacts/screenshots/after.png
```

The command takes exactly two file paths and produces no output when comparison
succeeds or the pixels differ. It only reads the input files.

| Exit status of the binary | Meaning |
| --- | --- |
| `0` | Image bounds and decoded RGBA pixels are identical. |
| `1` | Image bounds or pixels differ. |
| `2` | Wrong arguments, unreadable files, invalid PNGs or exceeded input limits; a diagnostic goes to stderr. |

`go run` reports a failing child as `exit status N` and returns its own nonzero
status. Build and invoke the binary directly if a script needs to distinguish
exit `1` from `2`.

Each PNG is limited to 64 MiB of encoded data, 16,384 pixels per dimension and
32 Mi pixels total before decoding. Comparison has no tolerance, perceptual
scoring or diff-image output.

See [screenshot capture](../../docs/design/screenshot-capture.md) for producing
consistent input captures and [branding](../../docs/design/branding.md) for
canonical artwork.
