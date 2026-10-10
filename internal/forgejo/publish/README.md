# Validated conditional publication

`publish` is Soda's narrow, unprivileged Git candidate validator and native
publication adapter. It is an internal Go package consumed by
`forgejo.Publisher`; operators configure that integration and repository users
see its resulting branches and pull requests in native Forgejo.

## Configure the consumer

`Config` requires an absolute private workspace root, a native repository
remote, publishing username and absolute restricted token file. The root must
be a directory and the token a regular file; both reject symlinks and group/other
permissions. Production remotes require HTTPS without embedded credentials;
HTTP is admitted only for local fixtures. The runtime needs native Git.

`Request` carries the recorded `factory.Run`, admitted base SHA, exact fresh
candidate commit, bundle bytes and known protected credential literals. Only
the coding role may publish, and bundles are bounded to 4 MiB.

## Validate before publication

The ordinary parent adapter uses:

```go
validated, err := cfg.PrepareValidated(ctx, request)
if err != nil {
    return err
}
defer validated.Close()
```

`PrepareValidated` imports the bundle into a fresh bare repository under the
private root. It checks the reported HEAD, bundle validity, ancestry from the
admitted input, protected path changes and credential literals in reachable
new Git objects. Hooks and inherited Git configuration are disabled. `Close`
removes that workspace. `ValidateCandidate` provides validation with immediate
cleanup and performs no push.

The protected paths include `.forgejo/`, `.github/workflows/` and configured
additions. Credential scanning includes historical objects and commit messages,
with limits of 10,000 objects, 4 MiB per object and 32 MiB total. Exceeding a limit
denies validation. Literal scanning does not prove absence of encoded secrets.

## Persisted native operations

Consumers supply one shared `BackgroundOperations` transport. `ObserveForPublish`
brackets exact branch tips with an idle native revision. `SubmitPublish` and
`SubmitPRCreate` submit stored operation identities with actor, repository,
authority revision, expiration and expected ref values. `ValidatedRepo.PushBranch`
pushes its candidate only under a registered branch operation.

Lookups and receipt decoders establish the native result; loss of a submit or
push response requires reconciliation. `Refusal` is a terminal verdict and
`Wait` means no verdict is available yet. The coordinator owns durable intent
and retry decisions; this package does not open a ledger or admit factory work.
Validation alone has local workspace effects; publication can change native
branches and create a PR.

See [the parent adapter](../README.md),
[factory coordinator](../../factory/control/README.md),
[factory interfaces](../../../docs/architecture/factory-interfaces.md), and
[trust model](../../../docs/architecture/trust.md).
