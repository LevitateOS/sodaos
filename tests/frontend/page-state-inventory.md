# Native/Soda page-and-state inventory (NFX-I02)

Isolated subjects only: synthetic fixture accounts/projects, the task-owned
native host socket and private test databases. No human session, Project or
terminal content is a fixture. Secrets and terminal content are masked in
every receipt; only IDs, states and masked markers are recorded.

| Page / state | Subject | Owner | Evidence driver |
| --- | --- | --- | --- |
| Native session generation (`/-/native/session`) | Fixture user `user2` | Fountain native | `soda_extension_terminal_live_test.go` |
| Workspace panel, new terminal (`create`) | Synthetic project `p…3`, login `soda-tester` | Soda terminal (D) | Fountain live test; `tests/frontend/terminal.test.ts` |
| Workspace panel, attach existing (`attach`) | Same synthetic project | Soda terminal (D) | Fountain live test; `terminal-stream/retirement` tests |
| Terminal ready/attached polling | Reserved session ID (hex) | Soda terminal (D) | Fountain live test state loop |
| Logout revocation closes socket | Fixture user session | Fountain native authority | Fountain live test `waitClosed` |
| Membership removal closes socket | Fixture membership row | Soda Store/API | Fountain live test |
| End terminal, then absent | Reserved session ID | Soda terminal (D) | Fountain live test; `terminal-end.test.ts` |
| Inert Reserve boundary (no host socket) | No native subject | — | Skipped, never passing preservation evidence |
| Browser fixture mount (synthetic API/renderer) | Fake socket/renderer in Chromium | Soda fixture (I) | `tests/frontend/fixtures/terminal-{fixture,driver}.ts` |

Rules: skipped or inert paths never count as preservation; a preservation
claim requires an actual renderer and socket object plus observed output.
Installed page/process coverage belongs to NFX-I08 against this inventory.
