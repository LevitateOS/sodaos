# Project services

Native nested workloads inside a project: databases and application containers run
through the project's workload engine, not the appliance engine socket.

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
