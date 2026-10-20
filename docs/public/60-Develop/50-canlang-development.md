# Develop Canlang applications in a Project

The Project includes Canlang's compiler, platform tools, runtime, and example testkit. Use them to compile an application, run a private development preview and examples, and connect an editor over standard LSP.

## Check and compile an application

Work from an application directory containing `app.can`. These commands check and compile its source, start a private preview, run checks and examples, expose the language server, then stop the preview:

```sh
can --version
can check app.can
can compile --format=json app.can > app.json
can dev start --source app.can
can dev check
can dev preview.open
can dev example.run --expected-revision r1
can lsp
can dev stop
```

Use the revision returned by the preview when it differs from `r1`. Example execution refuses an empty suite. Canlang's LSP provides diagnostics, completion, navigation, and formatting over standard LSP; configure your editor to connect to the `can lsp` process through its LSP integration.

On success, compilation writes `app.json`, checks report their result, and the preview exposes its current revision and private development app. Examples report their execution results; `can dev stop` stops this development preview while its retained development data remains separate from production.

## Keep development state separate

Compiler and platform tools are installed together. The preview keeps development state under your account, separate from production application data and identity. The ordinary `can-dev` preview covers D1 and Identity.

To develop the full selected Canlang profile, use the installed `can-soda` target with a separate development data directory, development owner bootstrap, and an HTTPS origin or proxy for Files. Never point development or example execution at production credentials or storage.

Canlang refuses source or schema changes that cannot safely use retained development state. Choose a fresh development data root when you intentionally need clean development state; this does not migrate production state.

## Maintain installed Project tools

Project tools persist in the retained Project root. Opening a terminal or importing an application image does not refresh them. Stop active Canlang processes before requesting a supported [same-root Project maintenance](../70-Operate/30-project-maintenance.md) operation.

## Continue to application operations

Packaging, app-local accounts, independent installation, and lifecycle belong to [Private applications](../70-Operate/40-private-applications.md). Private DNS and TLS are covered by [Private networking and TLS](../30-Set-up/20-private-networking-and-tls.md); stopped backups and fresh restores belong to [Backups and restoration](../70-Operate/60-backups-and-restoration.md).
