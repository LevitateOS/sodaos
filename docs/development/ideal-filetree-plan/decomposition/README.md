# Decomposition index

R02 note @HEAD `cb221d9c`: seam citations are audit-pinned like the slice
catalog; moved-owner paths (`rust/soda-host`, `rust/soda-release-*`,
`rust/identity-providers`, `appliance/`, `project-os/`) are intentionally
NOT path-substituted (splits shifted spans; see coverage-map STALE
banners). Current locations live in the coverage inventory/maps R02 notes;
successor dispositions for retiring sources stay in the seam entries.

The selective reconciliation at source `72e4bb9015b6d6a622b45638104c74851a137473`
updates affected host, Project, identity, server/state, install, release and
verification entries against the [library-adoption chapter](../library-adoption.md).
Their historical observed sizes/spans remain evidence, while explicitly
superseded pending engine splits no longer prescribe implementation. Remaining
domain/policy seams and library adapters are the target; completed moves stay
complete. The [task list](../implementation-tasks.md) and
[lane schedule](../implementation-lanes.md) remain the only dispatch queue.
A complete decomposition/tree inventory refresh waits for implemented boundaries.

## Detailed decomposition

The following names describe existing concern seams. The Rust host, installer,
importer and acceptance entries are the retained plan under the decided language
policy. Retiring Go/Python sources have removal dispositions and successor
references instead of target files for obsolete implementations. Historical
mockups are design material, not production owners. Embedded Rust tests stay
descendants of their implementation module so private access can survive. A
stateful frontend extraction requires an actual private seam; moving methods
across files alone cannot preserve one TypeScript class owner.


These reviews retain the structural baseline recorded in the main index, except
the identity broker and ST15 native-fixture entries explicitly refreshed at
0d8d3b8e. The host factory allocation in port-assessment.md is also refreshed. Page
groups organize existing source paths for navigation; they do not decide target
package boundaries. Retired sources keep their removal dispositions. Current
mixed-file resolution lives in the [responsibility maps](../coverage/maps/README.md).

| Source component group | File reviews |
| --- | ---: |
| [Browser and design](browser-and-design.md) | 13 |
| [Verification and support](verification-and-support.md) | 28 |
| [Factory coordination](factory-coordination.md) | 23 |
| [Server and state](server-and-state.md) | 13 |
| [Host runtime](host-runtime.md) | 23 |
| [Release production](release-production.md) | 48 |
| [Project runtime](project-runtime.md) | 8 |
| [Identity brokering](identity-brokering.md) | 10 |
| [Installation and operations](installation-and-operations.md) | 24 |
