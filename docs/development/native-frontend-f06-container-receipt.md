# NFX-F06 container development observations (post-A04)

Observed 2026-10-13 against Soda `72284a49` and Fountain `042b3bc325`.
Harness: `tests/frontend/native-container-geometry.test.ts` serves Fountain's
own `web_src/css/modules/container.css` and `web_src/css/features/extensions.css`,
container markup transcribed from Fountain
`templates/extensions/{page,panels}.tmpl` and `templates/base/{head,footer}.tmpl`,
and the body flex column from Fountain `web_src/css/base.css`.
This is a local fixture with transcribed markup, not native template rendering
or installed evidence. CDP changes CSS allocation and DPR; it does not set actual
browser zoom. The dimensions below remain useful constrained-allocation checks.
The fixture does not supply a consumer host-height adapter.

## Page container — requalified

Global Spaces page shape: `nav#navbar` + `.full.height` >
`.page-content.extensions > .ui.container > .extension-page >
`.extension-page-mount[data-extension-page]`, branded `components.css` last.

| Case | Observed | Result |
| --- | --- | --- |
| 1440x1000 @1x | container 1280x612, mount 1280x500, terminal 1018x380, horizontal overflow 0, mount inline height `""` | Workspace binds to the native 1280px container cap; F05 geometry holds |
| 1440x1000 @4x (360x250 CSS, DPR 4) | terminal height 55px, horizontal overflow 0; New terminal / Project settings reachable | Wave-1 F05 zoom fixes hold inside the native container |

## Panel container (A04/H-C) — requalified

Workspace panel shape: `body` (flex column) > `.full.height` (content,
`flex-grow: 0` via A04) + `aside#extension-panels.extension-workspace-panels`
(`flex: 1`, 24rem floor; 18rem under 768px) >
`section.extension-workspace-panel` > `[data-extension-panel]`, rendered
by `base/footer.tmpl` after `.full.height`.

| Case | Observed | Result |
| --- | --- | --- |
| 1440x1000 @1x | aside 1440x932, section 1440x932, mount 1408x870, Soda root inline height `""`, horizontal overflow 0 | Panels home fills the remaining viewport column; mount host-sized |
| 1440x1000 @1x, Sessions > Build | mount 870px, terminal 781px, `connected: true`, inline height `""`, horizontal overflow 0 | Terminal connects with usable height in host space; no invented Soda height |
| 1440x1000 @4x (360x250 CSS, DPR 4) | aside 360x288, mount 336x234, terminal 137px, `connected: true`, horizontal overflow 0; Sessions / New terminal reachable | Compact 18rem floor keeps a usable terminal at zoom 4x |

## Settings in panel container — requalified

Drawer/settings fixture inside the same host-sized panels-home mount.

| Case | Observed | Result |
| --- | --- | --- |
| 1440x1000 @1x | aside 1440x932, mount 1408x870, inline height `""`, horizontal overflow 0 | Settings mount host-sized; no invented Soda height |
| 1440x1000 @4x (360x250 CSS, DPR 4) | Overview 789px, Access 1014px, Network 483px, Factory 3191px tab-panel heights, horizontal overflow 0, first control of every tab reachable | F05 settings geometry holds in the final container |

## Remaining qualification

The three cases passed after the consumer adapter removal on 2026-10-13:
Workspace page, connected terminal panel and settings panel. They cover the
actual Fountain styles with transcribed container HTML at ordinary and reduced
allocation. They do not requalify creation/dialogs in native-rendered containers,
actual browser zoom, terminal object lifetime or installed behavior. Those joins
remain with F05/F06 and I04/I06/I08; the prior local responsive/focus checks retain
their original scope.
