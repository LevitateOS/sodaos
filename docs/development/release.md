# Release development workflow

How developers produce and qualify release candidates. Durable model:
[Release architecture](../architecture/release.md). Tool contracts:
[Native support](native-support.md).

## Pipeline map

| Symptom or task | Open first | Then |
| --- | --- | --- |
| Candidate will not build | `release/build`, `tools/soda-build` | `release/image` |
| Media/assemble wrong | `release/image` | `internal/installer` |
| Guest/fixture will not qualify | `release/qualify` | `internal/acceptance` |
| Sign/publish | `release/deliver` | `tools/soda-release` |

## Development vs qualification

- Use the smallest applicable development target while iterating.
- Do not relabel a failed release run as successful.
- Retained artifacts may support explicitly non-qualifying development checks.
- Run full production qualification when its prerequisites are demonstrated, not as
  the default debug loop.

## Commands

Follow the admitted interfaces in [Native support](native-support.md) for
candidate production, media, export and evidence. Publication, disk writes and
provider mutations need explicit approval.
