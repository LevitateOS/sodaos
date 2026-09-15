# Testing and native validation

How to validate Soda changes. Product journeys and native proof requirements live
here; tool transport lives in [Native support](native-support.md).

## Layers

| Layer | What it proves |
| --- | --- |
| Source checks | Go tests, TypeScript typecheck/lint, focused frontend suites |
| Authored browser fixtures | UI contracts against local/fixture servers |
| Native stage checks | `scripts/check-native.sh` against a prepared matching-architecture stage |
| Installed journeys | Real appliance/project behavior on an authorized target |

Cross-compilation or emulation is not native proof. x86_64 and aarch64 evidence
are independent.

## Source checks

Choose checks for the change. Common entry points are listed in
[AGENTS.md](../../AGENTS.md#commands). Examples:

```sh
go test ./internal/project ./internal/web/...
bun run typecheck
bun run test:frontend
bun run check:source
```

## Native stage

```sh
bash scripts/check-native.sh ARCH /ABS/PATH/TO/stage
```

Stage preparation and candidate production: [Native support](native-support.md).

## Product journeys to cover when touching those areas

- Project foundation: join, terminal, files, Git, shared tools, nested service
- Spaces: create/key/join, drawer, coordinated logout
- Managed terminal: reserve/create/exact attach/End, reload survival
- Operator runners: list/register/start/stop with operator gate
- Operator Tailnet: host settings and project enrollment policy
- Installation media and activate flows when those owners change

Exact script names and fixtures live under `tests/` and `scripts/`. Prefer extending
existing drivers over inventing new harnesses.

## Evidence rules

- Distinguish authored checks, local tests, native builds and installed evidence.
- Preserve credentials, unrelated work and failed evidence unless cleanup is approved.
- A passing receipt supports its stated scope only.
