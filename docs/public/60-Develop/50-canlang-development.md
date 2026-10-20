# Canlang development

The Project includes the Canlang compiler, platform tools, runtime and example testkit. Use them to check and compile an application, run a retained private development preview, execute its examples and connect an editor through the standard language server protocol (LSP).

Work from an application directory containing `app.can`. For example:

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

The compiler and platform tools are installed together. The preview keeps development state under your account, separate from production application data and identity. Use the revision reported by the preview when it differs from `r1`; example execution refuses an empty suite. The ordinary `can-dev` preview covers D1 and Identity. For development of the full selected Canlang profile, use the installed `can-soda` target with a separate development data directory, development owner bootstrap and an HTTPS origin or proxy for Files. Never point development or example execution at production credentials or storage.

Canlang's LSP provides diagnostics, completion, navigation and formatting over standard LSP. Configure your editor to connect to the `can lsp` process using its supported standard LSP integration. Canlang source or schema changes that cannot safely use retained development state are refused; choose a fresh development data root when you intentionally need clean development state. This does not migrate production state.

Project tools persist in the retained Project root. Opening a terminal or importing an application image does not refresh them. Stop active Canlang processes before requesting a supported [same-root Project maintenance](../70-Operate/30-project-maintenance.md) operation.

This page covers development state and tools. Packaging, app-local accounts, independent installation and lifecycle are owned by [Private applications](../70-Operate/40-private-applications.md). Private DNS and TLS are covered by [Private networking and TLS](../30-Set-up/20-private-networking-and-tls.md), and stopped backups and fresh restores by [Backups and restoration](../70-Operate/60-backups-and-restoration.md).
