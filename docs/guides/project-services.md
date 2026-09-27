# Project services

Native nested workloads inside a project: databases and application containers run
through the project's workload engine, not the appliance engine socket.

These are persistent human development services. Factory workspaces do not inherit
their sockets, mounts, credentials or lifetime. Run-owned resources follow the
separate [factory interface](../reference/factory.md) and cleanup boundary.

Authority and persistence: [Project OS](../reference/project-os.md).
OS role vocabulary: [Host strategy](../research/host-strategy.md).

## Boundary

- Project root/wheel administers the shared nested engine.
- Ordinary members consume service endpoints.
- Nested workloads have their own volumes and data; terminal End does not stop them.
- Compose and systemd units inside the project are ordinary extension points.

## Ordinary use

Keep service definitions in the project. Coordinate ports within the project rather
than assuming an appliance-wide port pool. Application-consistent backup and restart
are separate operations from terminal reconnect.

Related: [Develop](develop.md).

## Muse opt-in

Use a compatible Linux container with the architecture and runtime prerequisites
for the pinned Muse CLI. From project root, opt one service in for a provisioned
Soda account:

```sh
sudo soda-muse-compose --login soda-tester --file compose.yml --service development
```

The helper adds read-only tools, the launch interface and a private runtime
credential mount. It registers the actual immutable child container incarnation
against the authorizing account. The service keeps its configured user and volumes.
Run `muse` normally inside the service. Cloning or restarting the child requires
fresh validation through the helper; an old registration cannot authorize another
incarnation. Provider custody and credential exposure are owned by
[Credentials](../reference/credentials.md#normal-muse-command).
