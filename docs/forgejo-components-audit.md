# Forgejo presentation component audit

Audited the custom templates, every linked component/page stylesheet, guest theme
script/tests and their native stock **15.0.7** callers. The native installed theme
adapters and their separate staging path were also checked for ownership. Artwork,
backend business logic, Cockpit's separate frontend and the unimplemented drawer
are outside this presentation audit.

The three presentation partials are appropriately small. The first extraction's
CSS boundaries were incomplete: the controls file mixed toolbar and form work,
the base file mixed shell and content components, and page rules still duplicated
common decisions. This audit corrects those issues; it does not introduce Lit or a
new rendering framework.

## Findings and corrections

| Finding | Correction |
| --- | --- |
| One base stylesheet owned shell, intro, list, pagination and empty content. | Separate shell/tokens, intro, list and empty owners. The [composition contract](../appliance/forgejo/README.md#presentation-component-contract) identifies their exact inputs. |
| Toolbar rules also owned form controls and matched any native secondary navigation. | Forms own their controls; toolbar composition, tabs and explicit native context adapters have separate scopes. |
| Search button and hover selectors reached the nested native syntax dialog. | Select only direct search-input buttons and explicit action/trigger elements. In the old version, Forgejo's reparent-on-open behavior masked the dialog collision; an opened broken dialog was not observed. |
| Explore copied upstream tab visibility, destinations and overflow behavior. | A presentation wrapper calls native `explore/navbar` with its original context. |
| Notification lists had a second contract where the list itself carried `.soda-list`. | Every caller now wraps a direct native list. Notification IDs, row forms and replacement root remain intact. |
| Filled actions used light blue in dark mode; toolbar primary styling mixed neutral and filled states. | Theme-aware action/on-action tokens and an explicit filled toolbar modifier own default, hover and pressed states. |
| Guest palette aliases, shell rules and focus behavior were repeated. Explorer borrowed toggle dimensions from home CSS. | One guest theme/shell owner; toggle owns its own dimensions and states. Login retains its positioning and larger size. |
| Login drew an arrow using the native submit button's loader pseudo-element and hard-coded its error border. | Remove the pseudo-element decoration and use the native error border token. Native loading/disabled/error behavior remains upstream-owned. |
| Page files repeated layout, panel radius, focus and mobile decisions; dashboard heading styles reached rendered feed content. | Remove duplicate declarations, use shared dimensions and scope dashboard headings to authored heading/discovery areas. Native empty feedback belongs to the empty-state adapter. |
| Guest preference listeners ran on signed-in and unrelated native routes. | Native route/authentication flags gate the head script to anonymous pages with a guest toggle. Scoped CSS remains in one explicit asset registry. |
| Composition tests largely checked string presence. | Add executed caller/branch tests for native delegation, visible empty-state actions and guest script/markup boundaries. These complement, not replace, upstream compatibility and browser checks. |

## Boundaries intentionally retained

- `page_intro` owns presentation values only. It has no route dispatch, permission
  checks or arbitrary HTML slots. The compact variant covers an actual caller.
- `empty_content` owns its heading, optional icon and supporting copy. Native
  callers decide whether results are empty and which links/actions are available.
- Form sections remain semantic fieldsets around native fields. No dynamic form
  schema, generic card renderer, copied validation or new form controller is needed.
- The toolbar and form CSS adapt different native structures. They share semantic
  colors and dimensions, while retaining appropriate search, form and action sizes.
- Home's marketing layout, login's authentication wrapper, dashboard's native Vue
  repository widget, milestone progress cards and notification row actions have
  real, distinct responsibilities. Forcing these into a single generic component
  would introduce variants and obscure their native owners.
- `.soda-page` activates a full-page shell. It must not be reused as a drawer root.
  Future interactive components need local scope and native integration of their own.
- `forgejo/css/theme-soda-*.css` and `css/soda-controls.css` are native account-theme
  adapters shipped by appliance staging. The preview component styles/templates are
  a separate delivery surface that is not yet staged. Similar semantic values do
  not make either path dead code; both consume the canonical value-only palette.
- Global scoped CSS registration is retained to avoid another route/style dispatch
  table. It has a loading cost, but does not authorize components to style unrelated
  markup. Conditional bundling/loading is not needed to fix the component boundaries.

## Validation and limits

Validation results are recorded in the [current handoff](implementation-status.md#shared-forgejo-presentation-components).
Template render tests exercise Soda's composition boundary with native seams
stubbed. They are not a substitute for Forgejo's own server-side authorization,
permission and workflow tests. Browser checks use the existing local fixture data;
this audit does not claim production staging, deployment or complete upgrade proof.
