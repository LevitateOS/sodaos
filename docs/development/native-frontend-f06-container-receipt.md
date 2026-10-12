# NFX-F06 native-container requalification receipt (partial, A04-pending)

Observed 2026-10-13 against Soda `6a931fca` and Fountain `70f3f656c6`.
Harness: `tests/frontend/native-container-geometry.test.ts` serves Fountain's
own `web_src/css/modules/container.css` and `web_src/css/features/extensions.css`
and container markup transcribed from Fountain
`templates/extensions/{page,panels}.tmpl` and `templates/base/footer.tmpl`.
No Soda CSS change was needed or made; missing host space is not repaired
in Soda.

## Page container (available now) — requalified

Global Spaces page shape: `nav#navbar` + `.full.height` >
`.page-content.extensions > .ui.container > .extension-page >
.extension-page-mount[data-extension-page]`, branded `components.css` last.

| Case | Observed | Result |
| --- | --- | --- |
| 1440x1000 @1x | container 1280x612, mount 1280x500, terminal 1018x380, horizontal overflow 0, mount inline height `""` | Workspace binds to the native 1280px container cap; F05 geometry holds |
| 1440x1000 @4x (360x250 CSS, DPR 4) | terminal height 55px, horizontal overflow 0; New terminal / Project settings reachable | Wave-1 F05 zoom fixes hold inside the native container |

## Settings in panel container (available now) — requalified

Drawer/settings fixture inside the real panels-home mount at zoom 4x
(360x250 CSS): Overview 789px, Access 1014px, Network 504px, Factory
3230px tab-panel heights, horizontal overflow 0, first control of every
tab reachable. Panels are content-sized and the document scrolls; no
Soda inner-height repair was added.

## A04-dependent remainder (precise, blocks F06 Finish)

Ordinary native panel shape: `aside#extension-panels.extension-workspace-panels`
> `section.extension-workspace-panel` > `[data-extension-panel]`, rendered
by `base/footer.tmpl` after `.full.height`.

Observed at 1440x1000 @1x: aside 1440x111, section 1440x111, mount
1408x49, Soda root inline height `""`. Opening Sessions > Build connects
the terminal (`connected: true`) but its height is 0px: the panels home
supplies no definite block size (Fountain `extensions.css` gives only
`min-width: 0` on the home and `flex: 1; overflow: auto` on the mount,
which cannot resolve against an auto-height parent), so the
`height: 100%` workspace chain collapses.

Required from A04/H-C (Fountain, not Soda): a definite responsive block
size for `.extension-workspace-panels` / its panel sections on workspace
documents, with the panel heading accounted for, so the existing mount
rule yields a usable terminal height (>= 44px at zoom 4x). The F06 panel
test asserts this remainder (`mountH < 200`, `terminalH < 44`) and is
expected to fail when A04 lands; then re-run it, record the new
dimensions, and complete the settings/creation/dialog/compact composed
receipt for F06 Finish. Installed scope remains I08.
