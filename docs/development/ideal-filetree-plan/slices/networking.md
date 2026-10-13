# Networking

[Slice catalog and evidence scope](README.md). The cards below span current
source owners; they do not create packages, services or an approved intended model.

## First-principles refresh (2026-10-13)

Baseline: `fde19669440455a77e4c81c4f08ccc60a72276eb`. This is a bounded
current-owner/caller refresh using the networking contract and the historical
records below; it does not repeat their complete source reviews or advance their
native evidence. **Met** means the stated source model is supported; **partial**
means an explicit end-to-end gap remains; **unknown** is an unobserved outcome,
not a defect. Go serves/adopts domain requests, Rust performs host effects and
TypeScript presents them. GUIDANCE-01/03 retain those boundaries. No new network,
proxy, name service or automatic replay is proposed.

| Slice / purpose | First principles | Current assessment and evidence | Desired gap | Dependency-ordered implementation / check |
| --- | --- | --- | --- | --- |
| N01 Private origins, TLS and activation: establish private browser access. | Admit an explicit origin, private bind, operator identity and TLS choice before effects; client trust requires client evidence. | **Partial.** `cmd/soda-activate/src/activation.rs` validates root, TLS alternatives, private IP, configuration and fixed listeners before service effects; `system/host/config/proxy.Caddyfile` owns proxy wiring. Actual trust/reachability remains **unknown**. | Keep one commissioned origin and truthful failure after partial activation; demonstrate the existing client journey. | After O02, retain activation tests, inspect actual service/TLS outputs in a selected native fixture, then verify a trusted intended client. No rebuild just to test configuration logic. |
| N02 Project LAN access: expose usable native connection facts. | Membership admits disclosure; an observed address is a fact, not a route guarantee; native SSH/services remain their owners. | **Partial.** `internal/web/api/environments_api.go` delegates authorized connection reads to host; `lib/host/src/net.rs` admits configured subnet addresses. Client routing is **unknown**. | Demonstrate the ordinary `user@project-ip` journey without custom routing authority. | After P02–P04/P11, run the existing developer-access checks against the exact native Project and a selected client; retain address and route evidence separately. |
| N03 Host Tailnet control: administer the host's native identity/preferences. | Current configured operator and confirmation precede effects; uncertain command/readback stays uncertain; browser concealment must retire secrets/reads. | **Partial.** `internal/web/api/tailnet.go` checks operator/current session before and after host calls; `lib/host/src/tcontrol.rs` and `frontend/tailnet/soda-tailnet-page.ts` retain separate native and presentation owners. Installed availability is **unknown**. | Qualify current operator admission and readback after both successful and unconfirmed native actions. | After G01/H01/N01, exercise existing API refusal tests, then scoped native action/readback. Inspect native state before retrying an unconfirmed effect. |
| N04 Project enrollment policy: retain operator-approved policy and restricted provider inputs. | Projects inherit policy, never host identity; policy intent, provider admission and enrollment are separate facts; secrets stay private. | **Partial.** API clears `ClientSecret`; `lib/host/src/tcontrol_policy.rs` retains revisions and protected storage; `tcontrol_enroll.rs` distinguishes submitted enrollment from confirmed reachability. Real provider admission is **unknown**. | Qualify the selected policy/provider contract without adding a provider state machine. | After N03/H04, use existing save/check/rotate/disable refusal vectors; only then exercise the owner-selected provider and verify stored policy plus IDs-only observations. |
| N05 Project Tailnet selection: admit explicit networking intent for an original Project. | Product authority must establish who may select/change intent; current handler checks alone cannot grant it. Bind admitted mutation to current revision/original container without silent retargeting. | **Partial.** `internal/web/api/tailnet.go` currently distinguishes human-owner creation options and existing Project authority; `lib/host/src/tcontrol_policy.rs` owns bound intent. Selector authority and uncertain-creation replay remain **unknown** under REQ-Q1/N05-Q4, as does native behavior. | Resolve the selector/creation-replay contract before semantic changes; preserve existing revision/container refusals and independent LAN operation. | After P01/P03/N04, retain current refusal tests. Gate selection/replay changes on REQ-Q1; add neither replay ledger nor broader authority by inference. Native N06 evidence cannot settle this product decision. |
| N06 Project companion lifecycle: attach optional ephemeral Tailnet execution to one native run. | The companion belongs to an exact run incarnation; resolver/key cleanup cannot affect a replacement run or the persistent Project. | **Partial.** `lib/host/src/tailnet/companion/start.rs` gates on saved policy/image and waits for native runtime; `stop.rs` checks recorded Project/container/run identity. Installed resolver/logout behavior is **unknown**. | Verify start/stop/replacement against real Podman namespaces while preserving off-policy no-op behavior. | After P05/N04/N05, reuse incarnation/refusal fixtures; prove one enabled native lifecycle and one stale-run cleanup refusal before expanding scenarios. |
| N07 Git endpoint advertisement: refresh only an admitted native SSH listener. | Clone advertisement, browser origin and actual routing differ; confirm listener binding before an SSH-domain-only rewrite; restart only when needed. | **Partial.** Current owner is Rust `cmd/soda-forgejo-tailnet/main.rs` → `lib/host/src/tailnet/forgejo.rs`, with root/90-second admission, private inspection and conditional restart. The historical Go helper is retired; live route proof is **unknown**. | Qualify the current helper/listener/readback tuple; avoid planning another language cutover. | After N03/O04, run existing Rust listener/rewrite/refusal tests when this owner changes, then the installed advertisement driver for the exact configured private listener and intended Git client. |

The dependency order is O02/N01 and G01/H01 first, then N03/H04/N04,
Project authority and N05, and finally N06/N07 native journeys. The missing
native outcomes are evidence work, not authority to create new network services.
REQ-Q1 remains a separate owner-contract question for optional selection/replay;
current API behavior has not resolved it.
Independent A checked N02 address-versus-route evidence, N06 exact companion/run
cleanup and N07's current Rust entrypoint/caller. No blocking objection remained
within those source seams; this is not independent proof of all seven workflows.

Scoped upkeep at Soda `ac22514b` / Fountain `c72f5607b2` (2026-10-13): N03's
Tailnet entry/page consume the public host lifecycle, pausing reads and clearing
secrets on concealment, generation-checking resume and releasing local resources
on retirement. Configured-operator/product guards remain with their existing
owners. No network service, language or process moves; actual native/installed
operator availability and refusal retain their separate evidence gates.

## N01 Private origins, TLS and activation

Configure the appliance's explicit private browser origin, TLS material and activated service wiring.

- **Entrypoints:** soda-activate CLI; activate(); Caddy configured origin listener.
- **Owned data:** proxy.Caddyfile/proxy.env, TLS files and /etc/soda/activated; Native Forgejo origin-related configuration; Caddy data.
- **Authority:** Host operator/root commissioning; explicit client trust setup.
- **Dependencies:** [O02](operator-administration.md#o02-operator-identity-bootstrap); [N02](#n02-project-lan-access); [G01](forgejo-integration.md#g01-browser-authority-and-contributions); Native Caddy and systemd.
- **Source files:** `rust/soda-activate/src/main.rs:545 (historical source locator)`; `appliance/config/proxy.Caddyfile:6 (historical source locator)`; `appliance/services/soda-proxy.container (historical source locator)`.
- **Tests:** `rust/soda-activate/src/main.rs:1020 (historical source locator)` — Inline tests exercise activation against a fake system, TLS input refusal and root admission; [tests/build/proxy_image_test.go:10](../../../../tests/build/proxy_image_test.go#L10) — Structural image pin assertion only.
- **Unclear boundaries:** Activation coordinates several services; ownership of origin/TLS effects belongs here, while operator identity belongs to O02. Fake-system tests do not prove client trust or installed reachability.
- **Evidence status:** Current source mapped; correctness and installed behavior unreviewed.

- **Validity review:** [N01 audit record](../reviews/N01.md).

## N02 Project LAN access

Expose honest native Project connection information and verify ordinary private-network access.

- **Entrypoints:** GET /api/environments/{id}/connection; Host POST /connection; native container inspection; Installed developer-access probes.
- **Owned data:** Native Podman addresses, interfaces and service ports; Project.ip observation in Soda store; no alternate routing authority.
- **Authority:** Current Project member's connection access; host root configures host networking; Project administration controls native services.
- **Dependencies:** [P02](projects.md#p02-profile-and-runtime-readiness); [P03](projects.md#p03-human-membership-and-accounts); [P04](projects.md#p04-development-ssh-access); [P11](projects.md#p11-nested-services-and-volumes); Native Podman/SSH.
- **Source files:** [internal/web/api/environments_api.go:198](../../../../internal/web/api/environments_api.go#L198); `rust/soda-host/src/project.rs (historical source locator)`; `rust/soda-host/src/net.rs:66 (historical source locator)`; [docs/architecture/networking.md:5 (historical line locator)](../../../architecture/networking.md).
- **Tests:** `rust/soda-host/src/net.rs:84 (historical source locator)` — Inline IP/subnet admission vectors; no real route exercised; [internal/acceptance/developer_access_test.go:20](../../../../internal/acceptance/developer_access_test.go#L20) — Access-probe input/subnet validation unit tests; [tests/installed/project-os.sh](../../../../tests/installed/project-os.sh) — Installed Project access driver source; not executed.
- **Unclear boundaries:** Native address observation and input validation do not establish a working client route. Native workload ports/data remain P11's responsibility.
- **Evidence status:** Current source mapped; correctness and installed behavior unreviewed.

- **Validity review:** [N02 audit record](../reviews/N02.md).

## N03 Host Tailnet control

Observe and explicitly change the appliance's native Tailscale identity and preferences.

- **Entrypoints:** GET /api/settings/tailnet; POST /api/settings/tailnet/host; Host /tailnet/settings and /tailnet/host.
- **Owned data:** Native tailscaled device state/preferences; observed revision and addresses; Browser action drafts and bounded readback projections.
- **Authority:** Configured Soda operator with current native browser admission; root native helper.
- **Dependencies:** [G01](forgejo-integration.md#g01-browser-authority-and-contributions); [H01](shared-supporting-slices.md#h01-private-ipc-and-service-lifetime); [N07](#n07-git-endpoint-advertisement); Native tailscaled LocalAPI.
- **Source files:** [internal/web/api/tailnet.go:45](../../../../internal/web/api/tailnet.go#L45); `rust/soda-host/src/tcontrol.rs:197 (historical source locator)`; `rust/soda-host/src/tcontrol_native.rs:509 (historical source locator)`; [frontend/tailnet/soda-tailnet-page.ts](../../../../frontend/tailnet/soda-tailnet-page.ts).
- **Tests:** [internal/web/tailnet_test.go:147](../../../../internal/web/tailnet_test.go#L147) — API admission and failure-body secrecy using a stub host; `rust/soda-host/tests/tcontrol_oracle.rs:2387 (historical source locator)` — Scripted host-action/readback cycle and native fixture vectors.
- **Unclear boundaries:** Host device identity is separate from Project enrollment policy and companion identities. Native command/readback uncertainty must not be described as confirmed reachability.
- **Evidence status:** Current source mapped; correctness and installed behavior unreviewed.

- **Validity review:** [N03 audit record](../reviews/N03.md).

## N04 Project enrollment policy

Validity review: [N04 record](../reviews/N04.md). Its recorded scope and completion dimensions govern audit status.

Maintain operator-authorized standing enrollment policy and its restricted provider credentials.

- **Entrypoints:** POST /api/settings/tailnet/enrollment; Host /tailnet/enrollment; Control.enrollment().
- **Owned data:** /var/lib/soda-tailnet/policy.json: revisions, binding, tags, admission/default and restricted enrollment credential.
- **Authority:** Configured Soda operator via current session; root policy store; explicit native provider authorization.
- **Dependencies:** [G01](forgejo-integration.md#g01-browser-authority-and-contributions); [H01](shared-supporting-slices.md#h01-private-ipc-and-service-lifetime); [H04](shared-supporting-slices.md#h04-configuration-and-filesystem-primitives); Native Tailscale enrollment API.
- **Source files:** [internal/web/api/tailnet.go:86](../../../../internal/web/api/tailnet.go#L86); `rust/soda-host/src/tcontrol.rs:257 (historical source locator)`; `rust/soda-host/src/tcontrol_policy.rs (historical source locator)`; `rust/soda-host/src/tcontrol_provider.rs (historical source locator)`.
- **Tests:** `rust/soda-host/tests/tcontrol_oracle.rs:2462 (historical source locator)` — Policy save/check/rotate/disable and provider-key guards with fake transport; [internal/web/tailnet_test.go:222](../../../../internal/web/tailnet_test.go#L222) — Input secrecy and endpoint-override refusal.
- **Unclear boundaries:** These are device-enrollment credentials, not AI account custody. Policy intent, provider checks and a connected Project node are separate facts.
- **Evidence status:** Current source mapped; correctness and installed behavior unreviewed.

## N05 Project Tailnet selection

Validity review: [N05 record](../reviews/N05.md). Its recorded scope and completion dimensions govern audit status.

Record and admit explicit networking intent for one original Project/container.

- **Entrypoints:** GET /api/repositories/{repositoryID}/tailnet-options; GET/POST /api/environments/{id}/tailnet; Host /tailnet/project and /tailnet/policy.
- **Owned data:** project-{id}.json selected intent, revision and original container/policy binding; Creation selection and read-only browser projection.
- **Authority:** Current human repository owner selects creation options; existing Project changes require current administration or configured operator; member visibility is narrower.
- **Dependencies:** [P01](projects.md#p01-repository-association-and-creation); [P03](projects.md#p03-human-membership-and-accounts); [N04](#n04-project-enrollment-policy); [G01](forgejo-integration.md#g01-browser-authority-and-contributions); [H01](shared-supporting-slices.md#h01-private-ipc-and-service-lifetime).
- **Source files:** [internal/web/api/tailnet.go:114](../../../../internal/web/api/tailnet.go#L114); [internal/web/api/tailnet.go:146](../../../../internal/web/api/tailnet.go#L146); `rust/soda-host/src/tcontrol_policy.rs (historical source locator)`; [frontend/spaces/sodaspaces-network.ts](../../../../frontend/spaces/sodaspaces-network.ts).
- **Tests:** [internal/web/tailnet_test.go:250](../../../../internal/web/tailnet_test.go#L250) — Current-owner/member/organization-owner read and mutation matrix; `rust/soda-host/tests/tcontrol_oracle.rs:1251 (historical source locator)` — Project enable revision checks and refusal of implicit retargeting.
- **Unclear boundaries:** Creation selection, later administration and runtime enrollment have different authority/lifetimes. Repository transfer does not silently retarget native Project identity.
- **Evidence status:** Current source mapped; correctness and installed behavior unreviewed.

## N06 Project companion lifecycle

Validity review: [N06 record](../reviews/N06.md). Its recorded scope and completion dimensions govern audit status.

Bind optional Tailnet execution and ephemeral enrollment to the exact native Project run.

- **Entrypoints:** soda-host --tailnet-action run|stop --project ID; soda-tailnet@.service; Companion.start_tailnet()/stop_tailnet().
- **Owned data:** Run-incarnation metadata, companion identity, namespace/resolver bindings and temporary enrollment-key files; Native companion node state; /var/lib/soda-tailnet policy is consumed, not owned here.
- **Authority:** Root fixed native helper under explicit Project policy and configured companion image.
- **Dependencies:** [P02](projects.md#p02-profile-and-runtime-readiness); [P05](projects.md#p05-project-startstop); [N04](#n04-project-enrollment-policy); [N05](#n05-project-tailnet-selection); Native Podman/systemd/tailscaled.
- **Source files:** `rust/soda-host/src/tailnet_companion.rs:827 (historical source locator)`; `rust/soda-host/src/tailnet_runtime.rs (historical source locator)`; `rust/soda-host/src/tailnet_files.rs (historical source locator)`; `rust/soda-host/src/tcontrol_enroll.rs (historical source locator)`; `appliance/services/soda-tailnet@.service:10 (historical source locator)`.
- **Tests:** `rust/soda-host/src/tailnet_companion.rs:1310 (historical source locator)` — Inline fake-executor identity, resolver, start/stop and uncertain-logout tests; `rust/soda-host/tests/tcontrol_oracle.rs:2650 (historical source locator)` — Run enrollment identity/uncertainty fencing with fixture provider transport.
- **Unclear boundaries:** Code/configuration support does not establish an enabled or qualified installed companion. Lifecycle cleanup must respect native incarnation; it does not stop or delete the persistent Project.
- **Evidence status:** Current source mapped; correctness and installed behavior unreviewed.

## N07 Git endpoint advertisement

Validity review: [N07 record](../reviews/N07.md). Its recorded scope and completion dimensions govern audit status.

Refresh native Forgejo SSH advertisement only for an explicitly admitted private listener.

- **Entrypoints:** Tailnet host action refresh-forgejo; soda-forgejo-tailnet CLI.
- **Owned data:** FORGEJO__server__SSH_DOMAIN in forgejo.env; native service restart outcome; Observed native listener and Tailnet endpoint identity.
- **Authority:** Configured operator at API; host root at CLI/effects.
- **Dependencies:** [N03](#n03-host-tailnet-control); [G01](forgejo-integration.md#g01-browser-authority-and-contributions); [O04](operator-administration.md#o04-native-host-administration-and-updates); Native Forgejo/Podman/systemd.
- **Source files:** `rust/soda-host/src/tcontrol_native.rs:792 (historical source locator)`; `cmd/soda-forgejo-tailnet/main.go:32 (historical source locator)`; `internal/forgejo/tailnet.go (historical source locator)`.
- **Tests:** `cmd/soda-forgejo-tailnet/main_test.go:8 (historical source locator)` — Listener matching and secrecy unit test; no live restart; [tests/installed/forgejo-advertisement.sh](../../../../tests/installed/forgejo-advertisement.sh) — Installed advertisement verification driver source; not executed.
- **Unclear boundaries:** Keep native Git advertisement separate from browser origin/TLS and actual route proof. soda-forgejo-domain is a different operator recovery helper. The Go helper citation above is historical; the selected current implementation is [cmd/soda-forgejo-tailnet/main.rs](../../../../cmd/soda-forgejo-tailnet/main.rs) and [lib/host/src/tailnet/forgejo.rs](../../../../lib/host/src/tailnet/forgejo.rs). Its source cutover is complete, while installed/native qualification remains open per the [port assessment](../port-assessment.md#ownership-decisions-before-further-splitting).
- **Evidence status:** Current source mapped; correctness and installed behavior unreviewed.
