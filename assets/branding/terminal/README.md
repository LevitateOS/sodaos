# Terminal identity

The 32-column, 16-row ASCII mark samples the canonical offset-core SVG. `#` forms
the red outer plate; `@` forms the contrasting middle plate. Spaces keep the core
and background open. The two glyphs also distinguish the layers without color.
The grid assumes ordinary terminal cells roughly twice as tall as they are wide.

- `sodaos.txt`: fastfetch text logo, using its native `$1`/`$2` color placeholders.
- `fastfetch.jsonc`: red `#df001b` and default terminal foreground (`39`). A normal
  dark/light terminal theme supplies a contrasting white/black middle layer;
  there is no background fill or hard-coded white middle plate.
- `motd.txt`: equivalent plain ASCII plus the product name, with no escape codes
  or fastfetch placeholders.

Regenerate with `python3 scripts/render-terminal-logo.py`; verify with `--check`.
The generator reads the actual canonical SVG, including its even-odd core cutout.

Preview from the repository root:

```sh
fastfetch --config assets/branding/terminal/fastfetch.jsonc --logo assets/branding/terminal/sodaos.txt
```

The native image staging script installs the logo at
`/usr/share/soda/fastfetch/sodaos.txt`, the preset at `/etc/fastfetch/config.jsonc`,
and plain text at `/etc/motd`. Fastfetch's normal user-config precedence remains;
this change does not install the fastfetch executable, alter a developer's own
configuration, or add fastfetch calls to SSH command/SCP/SFTP streams.

Validated with fastfetch 2.68.1 using a logo-only `break` module, in color and
plain output. No host details are needed for the logo check. Native image
installation is separate from these source/staging and local rendering checks.

Reference: [Fastfetch logo options](https://github.com/fastfetch-cli/fastfetch/wiki/Logo-options).
