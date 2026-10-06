# SodaOS library reuse coverage ledger

Canonical report: [investigation report](library-reuse-investigation.md). Reviewed SodaOS `72e4bb9015b6d6a622b45638104c74851a137473`; relevant SDK `86a70e1155f1036fdcd38f2af49d4ca6defa8280`. This ledger records investigation coverage, not replacement acceptance or installed qualification.

## Source scope and reconciliation

The census has **1,412** tracked Go/Rust files: **852 production**, **16 production maintenance**, **1 compiled production test-helper**, **536 test**, **7 test-support**. The **869 production-bearing** files contain **161,562 physical lines**, including inline tests/comments. Primary screening pools: network **276**, crypto/formats **178**, persistence/generic **163**, coordinator **252**. Relevant SDK: **19 Go files**, **11 production / 8 test**.

All 1,412 files have one screening owner and an evidence-index record; every production-bearing file has a completed generic body screen. Named candidates have deeper defining-body/caller/test review as described in the report/indexes. Tests are inventoried; selected assertions were read, and remaining test/domain correctness is explicitly unclaimed. There are **zero unmapped production files, zero duplicate primary owners and zero unknown finding tags**. Cross-review records are supplementary evidence, not additional ownership.

Registration reconciliation excluded four cfg(test) Rust support modules (installer console/enroll/OCI and Muse maintenance) and the orphaned Go testoci helper from production. The original group dispatch counts therefore differ from final classes. Store identity fixture APIs are compiled in the production package but have only traced test callers; their separate class keeps that distinction visible. No tracked generated/vendored Go/Rust implementation was identified. Active maintenance, SQLite-native fixture/probe support and same-process rootfs fixture serving are not silently excluded as tests. Historical predecessor names in comments are not active owners.

## Mandatory starting areas

| Family | Supported verdict and retained boundary | Findings / primary owner | Independent challenge |
|---|---|---|---|
| HTTP | Replace complete Rust framing engines; preserve bounds, deadlines, listener authority, synchronous facade and no-replay policy. Retain Go net/http. Parser-only adoption is insufficient. | N1–5 / network | crypto, coordinator; provider distinctions reconciled |
| WebSocket | tungstenite with one nonblocking owner, wakeup and read-ahead; retain terminal/child/shutdown authority. Delete handshake SHA-1/Base64. | N6 / network; CF-01/04 deletion overlap | crypto |
| PostgreSQL | postgres complete sync driver; preserve typed parameters and transaction lifetime; explicit deadline/cancel/reconnect gate. postgres-protocol is only codecs/auth pieces. | PG01 / persistence | network corrected live-race overstatement |
| Hashes and curves | sha2 plus existing RustCrypto point validation; preserve canonical domain inputs. Go primitives retained. | CF-01/02/05/06 / crypto | persistence, coordinator |
| SSH | ssh-key format engine; all eight raw/eight certificate forms checked; preserve separate host/installer algorithms, options, limits and canonical/raw-byte contracts. | CF-03 / crypto | persistence |
| JSON | Serde engines with duplicate/depth/full-input and DTO profiles; preserve raw signed bytes. Go engine retained; SDK cap+one consolidation. | JSON01/02 / persistence; SDK01 / coordinator | network, coordinator, persistence |
| Base64 | Every independent codec profiled for alphabet/padding/CRLF/unused bits/errors; base64 0.22.1. No production URL-safe codec found. | CF-04 / crypto | persistence |
| URLs, IP and time | url/percent-encoding, std IP, time; retain lexical/admission/zone/mask/wire rules and shared Linux monotonic origin. | N7/8/9/13 / network; X50901 / coordinator | crypto, coordinator |
| SQL placeholders | Delete Go/Rust translation and use native PostgreSQL parameters; no ORM/parser. | SQL01 / persistence | network |
| RNG and temporaries | getrandom fails closed; tempfile with explicit creation permissions, rooted custody and publication/cleanup; delete unused temporary. | RNG01/TMP01/02 / persistence | crypto, coordinator |

## All findings and one primary owner

Each ID resolves to a report heading with exact defining files/callers, assumptions versus requirements, evidence/limits, retained adapter duties, migration checks, priority and estimated net removal/effort. RETAIN rows have no required adoption. The selected directions below are definite; CFG01 is the sole formal requirement blocker. PG01's selected driver still requires its stated cutover proof.

| ID | Primary | Disposition | Preferred owner or adapter | Independent challenge | Priority |
|---|---|---|---|---|---|
| N1 | network | REPLACE | hyper 1.12.0 + hyper-util 0.1.21 + Tokio 1.53.2 | crypto_formats | P1 |
| N2 | network | REPLACE | same Hyper owner with synchronous facade | crypto_formats | P1 |
| N3 | network | REPLACE | ureq 2.12.1 | crypto_formats | P1 |
| N4 | network | CONSOLIDATE | ureq 2.12.1; acceptance curl retained | persistence_generic | P2 |
| N5 | network | REPLACE | same in-process Hyper owner | coordinator | P1 |
| N6 | network | REPLACE | tungstenite 0.30.0, one nonblocking owner | crypto_formats | P1 |
| N7 | network | REPLACE | url 2.5.8 + percent-encoding 2.3.2 | crypto_formats | P1/P2 |
| N8 | network | REPLACE | std::net with zone/mask/admission adapters | crypto_formats | P2 |
| N9 | network | REPLACE | time 0.3.55; retain cross-process monotonic origin | crypto_formats | P1/P2 |
| N10 | network | CONSOLIDATE | rustix 1.1.5; retain audited SO_PEERPIDFD call | crypto_formats | P1/P2 |
| N11 | network | DELETE | active Muse owner | crypto_formats | P2 |
| N12 | network | RETAIN | explicit Soda policy | coordinator | P3 |
| N13 | network | REPLACE | url 2.5.8 plus conservative byte fallback | coordinator | P1 |
| N14 | network | RETAIN | std + serde_json 1.0.151; repair deadlines | coordinator | P1 |
| CF-01 | crypto_formats | REPLACE | sha2 0.10.9; SHA-1 deletion via N6 | persistence_generic | P1 |
| CF-02 | crypto_formats | REPLACE | p256 0.13.2 / p384 0.13.1 / p521 0.13.3 | persistence_generic | P1 |
| CF-03 | crypto_formats | REPLACE | ssh-key 0.6.7 with explicit algorithm/options policy | persistence_generic | P1/P2 |
| CF-04 | crypto_formats | REPLACE | base64 0.22.1 with distinct profiles | persistence_generic | P2 |
| CF-05 | crypto_formats | REPLACE | p256 0.13.2 + spki 0.7.3 / der 0.7.10 | persistence_generic | P1 |
| CF-06 | crypto_formats | REPLACE | ecdsa 0.16.9 typed signatures | coordinator | P1/P2 |
| CF-07 | crypto_formats | REPLACE | tar 0.4.46; delivery writer separately REL01 | coordinator | P2 |
| CF-08 | crypto_formats | CONSOLIDATE | shared bounded 64-byte accessor plus purpose gates | coordinator | P3 |
| CF-09 | crypto_formats | CONSOLIDATE | flate2 1.1.10 with explicit EOF/member policy | coordinator | P2 |
| PG01 | persistence_generic | REPLACE | postgres 0.19.14; operation-deadline cutover gate | network | P1 |
| SQL01 | persistence_generic | DELETE | native PostgreSQL parameters; existing pgx | network | P2 |
| JSON01 | persistence_generic | REPLACE | serde 1.0.229 + serde_json 1.0.151 | network | P1/P2 |
| JSON02 | persistence_generic | RETAIN | encoding/json plus boundary policy | network | P2 |
| RNG01 | persistence_generic | REPLACE | getrandom 0.4.3, fail closed | crypto_formats | P1 |
| TMP01 | persistence_generic | REPLACE | tempfile 3.27.0 with creation mode/custody | crypto_formats | P1/P2 |
| TMP02 | persistence_generic | DELETE | remove unused allocation and cleanup | coordinator | P3 |
| FS01 | persistence_generic | CONSOLIDATE | rustix 1.1.5 with explicit authority | crypto_formats | P1/P2 |
| PATH01 | persistence_generic | DELETE | std::path with admitted absolute-path policy | network | P2 |
| PROC01 | persistence_generic | CONSOLIDATE | std::process; delete compatibility diagnostics | coordinator | P1/P2 |
| FFI01 | persistence_generic | REPLACE | libc 0.2.190; retain NSS/signal policy | crypto_formats | P2 |
| FMT01 | persistence_generic | DELETE | Rust format_args! | coordinator | P3 |
| CLI01 | persistence_generic | RETAIN | explicit small parsers | coordinator | P3 |
| WALK01 | persistence_generic | REPLACE | walkdir 2.5.0 with authority gates | crypto_formats | P3 |
| CFG01 | persistence_generic | BLOCKED | native corpus needed; preferred rust-ini 0.21.3 conditional | coordinator | P2 |
| SQLITE01 | persistence_generic | RETAIN | modernc driver pending reachable consumer retirement | coordinator | P3 |
| RED01 | persistence_generic | REPLACE | aho-corasick 1.1.5 with bounded overlap adapter | network + coordinator | P1/P2 |
| CLI03 | persistence_generic | REPLACE | clap 4.6.7 + humantime 2.4.0 positive duration policy | network + coordinator | P2 |
| REL01 | coordinator | REPLACE | tar 0.4.46 deterministic adapter | network | P2 |
| REL02 | coordinator | CONSOLIDATE | existing delivery owner: tar/flate2/sha2 | network | P1/P2 |
| REL03 | coordinator | CONSOLIDATE | existing tar 0.4.46 + flate2 1.1.10 | crypto_formats | P2 |
| CLI02 | coordinator | REPLACE | clap 4.6.7 with value-aware aliases | network | P2 |
| XML01 | coordinator | REPLACE | roxmltree 0.21.1 + svgtypes 0.16.1 | crypto_formats | P2 |
| CFG02 | coordinator | RETAIN | small section/key set scanner | crypto_formats | P3 |
| PROC02 | coordinator | CONSOLIDATE | existing std-backed image runner, shared cancellation | persistence_generic + network | P1/P2 |
| SDK01 | coordinator | CONSOLIDATE | existing Go cap+one admission owner | persistence_generic | P2 |
| X50901 | coordinator | REPLACE | x509-cert 0.2.5 plus existing signature math | crypto_formats | P2 |
| KEEP01 | coordinator | RETAIN | existing standard and locked engines | all groups | P3 |
| SYS01 | coordinator | REPLACE | cargo metadata --format-version 1 --no-deps | crypto_formats | P2 |
| DEAD01 | coordinator | DELETE | remove unused test-support; retain testify | crypto_formats | P3 |

## Additional search and evidence limits

The search covered actual imports/definitions/body operations for protocol state machines, format lexers/binders/emitters, hash/field arithmetic, duplicate codecs, CLI/duration/config, escaping, compression/archives, ELF, Cargo discovery, randomness/temp, rooted filesystem, process/FFI/NSS, descriptor/peer and dead compatibility machinery. Pure domain predicates, template overrides, provider routing, standard-backed Go engines and meaningful authority wrappers were retained with reasons. Extra supported candidates include archive trailer/decoded-allocation issues, XML/SVG parsing, CLI emulation, Cargo target scanning, evidence matching/finalization, SDK input bounds and orphaned test support.

**Visible limits:** CFG01 needs the native effective-config corpus before final parser selection. PostgreSQL operation deadlines, X.509 critical-extension/Caddy fixtures, new dependency feature/license/cache closure, supported kernel APIs, replacements' full integration and native installed behavior remain acceptance work. Humantime 2.4.0 exact manifest/MSRV retrieval was unavailable; its older cached metadata is not treated as proof. None of these is hidden by a clean checkout or existing tests. No mandatory family or discovered production candidate is left without a disposition/primary owner.

Isolated executed probes cover only the questions stated in the report. Generic screening is not universal domain correctness. Per-reviewer evidence indexes retain exact scopes and selected-test gaps; RETAIN-prefixed policy tags are screening annotations rather than additional independent findings. Source /home/vince/Projects/output/library-reuse-investigation-20261006/source-coverage.csv references all cross-review evidence but contains only one primary screen row per file.

The report's final backlog and safe lanes define prerequisites without dispatching implementation or folder moves. Restructuring owners reached preserved safe checkpoints; timer stopped and command handles retained. Production source, dependencies and ideal-filetree inputs remain unchanged.

Evidence: [source census](../../../output/library-reuse-investigation-20261006/source-coverage.csv), [SDK scope](../../../output/library-reuse-investigation-20261006/sdk-coverage.json), [final verification](../../../output/library-reuse-investigation-20261006/final-verification.json), [baseline](../../../output/library-reuse-investigation-20261006/baseline.json).
