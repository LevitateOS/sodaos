# auth

Verify native extension authority and serve Soda profile resources for the
current Forgejo actor. This internal Go package supports
[`web`](../README.md) and [`api`](../api/README.md); it is an integration library
with no executable or independent browser login flow.

## Integrate

`auth.New(*config.Config, *store.Store)` constructs the profile `Service`.
Its `ExtensionHandler()` serves private session, preference and account-key
routes, normally reached through the product API's extension handler.

`ExtensionAuthority(request)` verifies one SDK context and admission, extension
ID `soda`, the matching HTTP action and the current native actor through a
bounded callback. `ExtensionContribution` admits the declared Spaces, Tailnet
and Workspace mounts; contribution-specific operator checks remain in the
extension process and product handlers.

Consumers must pass genuine native SDK authority from the private extension
runtime. Profile handling checks session generation, live authority and
same-origin JSON mutations. Browser IDs or an invented header cannot replace
native admission.

## Profile effects

Handlers upsert the native actor's Soda profile, read/update its display name,
and list/add/remove saved development-access public keys. They can also list
the actor's native Forgejo public keys through its current SDK authority.

`NormalizeDevelopmentKey` accepts one public SSH key without authorized-key
options and returns canonical public material plus its SHA-256 fingerprint.
Removing the final saved key requires explicit confirmation. Saved-key removal
does not revoke an already installed project SSH key; Forgejo Git keys and Soda
development-access keys have separate ownership.

## Shared HTTP helpers

`JSONResponse` and `JSONError` emit JSON responses; `ProviderError` maps native
failures without credential/provider response bodies. `AllowAPIMethod` checks
methods. Apply a request-body limit before `DecodeAPIObject`; native handlers
use `APIBodyLimit` (64 KiB) and strict single-object JSON decoding.
`PositiveID` and `ValidRepositoryPart` validate canonical identifiers.

[HTTP API](../../../docs/reference/api.md) owns route details;
[Forgejo customization](../../../docs/reference/forgejo.md) owns native browser
integration. Account access and persistence belong to
[Projects](../../../docs/product/projects.md), and authority to
[Trust](../../../docs/architecture/trust.md).
