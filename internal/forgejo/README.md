# Native Forgejo integration

`forgejo` connects Soda's backend to native Forgejo identity, observations and
conditional operations. It is an internal Go integration package; repository
users interact through native Forgejo and Soda's project/factory controls.
Forgejo owns its data and authorization rules. Soda uses supported HTTP and
extension interfaces rather than reading Forgejo's database.

## REST client

The dashboard constructs `forgejo.New(config.ForgejoInternalURL)`. A read of the
current token owner has this shape:

```go
client := forgejo.New(internalOrigin)
user, err := client.Current(ctx, token)
```

`internalOrigin` is the configured native origin, and `token` is loaded through
the caller's private credential path. `Current` calls `/api/v1/user`.
`RevokeCurrentToken` revokes that token; `OwnPublicKeys` reads the bound actor's
native public keys. The default client uses a 30-second timeout, refuses
redirect following and bounds decoded responses. Handle transport, HTTP and
invalid-response errors through the package's error vocabulary.

## Factory adapters

| Constructor | Consumer role |
| --- | --- |
| `NewServiceBackground` | Shared authenticated native service callback transport for revision reads, snapshots and persisted operations. |
| `NewServiceObserver` | Permission-checked issue/PR evidence and actor resolution for readiness and dispatch. |
| `NewPublisher` | Candidate validation, conditional branch publication and PR creation. |
| `NewReviewer` | Conditional native review operations. |
| `NewMerger`, `NewCheckAssessor` | Conditional merge and native check evidence. |

The web facade wires these into `factory/control`. Background integration needs
the configured Unix callback socket, expected native host peer UID and a
restricted credential file for the chosen actor. Review and merge credentials
are separately configured. Constructors defer I/O until use.

Share one `ServiceBackground` between the observer and operation adapters using
`observer.ShareBackground(background)`. Native bootstrap revokes the previous
service admission, so separate transports can invalidate each other's reads
and operations. Snapshot observations are bound to equal idle native revisions.

## Effects and outcomes

Reads do not authorize new work. Publication, review and merge adapters can
submit native mutations under persisted operation IDs and exact observations.
Lost replies are reconciled by lookup; unknown outcomes remain waiting or fenced
instead of being treated as success. Actor credentials stay in restricted files,
and candidate Git processing belongs to [publish](publish/README.md).

See [dashboard wiring](../web/server.go),
[Forgejo reference](../../docs/reference/forgejo.md),
[factory interfaces](../../docs/architecture/factory-interfaces.md), and
[trust boundaries](../../docs/architecture/trust.md).
