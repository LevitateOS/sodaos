# Create and manage repository Projects

A Project is a repository's persistent shared development environment. People and Factory work use its Rocky headless root with separate Linux accounts, checkouts, and execution identities.

## Create or reuse a Project

The human owner of a repository creates its Project from that repository's settings. Organization-owned creation is unsupported. To create or confirm the environment:

1. Open settings for the intended repository and inspect its Project state.
2. If no Project exists, create one there. Creation reserves and starts the shared Rocky headless environment; it does not Join anyone, clone the repository, or enroll an AI provider.
3. If creation reports an uncertain result, return to the same repository settings and inspect the original reservation. Retry only after confirming no Project was created; do not create a second environment while the first result is unresolved.

When creation succeeds, the repository has one persistent Project available for explicit human Join and later Factory preparation.

## Join the Project

Each person joins with the native Forgejo account that will own their access. To join:

1. Sign in to Forgejo as that person and open the intended repository's Project.
2. Choose **Join environment**. The account must currently have native code-write permission.
3. Choose account-only access or explicitly select public SSH keys if direct SSH is needed. A private key is never requested.

On success, the Project provisions that person's Linux account and records membership only after the native account operation succeeds. Account-only Join is supported, so browser development does not require SSH setup.

```mermaid
flowchart TD
    accTitle: Project creation and human membership
    accDescr: A human repository owner creates the persistent Project once. Creation does not Join anyone; each person needs current code-write permission and an explicit Join to obtain their own account.
    A["No Project yet"] --> B["Repository owner creates Project"]
    B --> C["Shared persistent root"]
    C --> D{"Code-write access?"}
    D -->|"Yes"| E["Person explicitly Joins"]
    D -->|"No"| F["Request access from owner"]
    E --> G["Own Project account"]
```

The diagram separates creating the shared root from granting each person access. [Start and Stop](#start-and-stop-the-retained-root) act on that same root; Start does not revive interrupted terminals or Factory work.

```doc-visual
{
  "placeholder": "Project Overview shows its running identity and the browser-only Join action for a write-authorized account.",
  "caption": "handbook-demo Overview: 1 the Rocky Project identity, 2 the Running state, and 3 Access: Not joined beside Join environment for the write-authorized account. Browser-only Join requires no SSH key.",
  "alt": "handbook-demo Project overview showing a running Rocky environment and visible Join environment action",
  "production": {
    "method": "screenshot",
    "instructions": [
      "Use an approved documentation instance with a real sample repository named handbook-demo and its already-created, running Rocky headless Project.",
      "Use the seeded sample account guide-writer with native repository code-write permission and no Project membership; keep its SSH-key selection empty and do not connect a provider.",
      "In Forgejo, navigate through the visible handbook-demo repository settings into Soda Project settings, then select the Overview tab; do not guess or type a deep route.",
      "Show the actual repository and Project identity, Running state, Access: Not joined status, and Join environment button; do not click Join or alter Project state.",
      "Record the actual SodaOS release, Fountain revision, and capture date in the repository asset notes without including host IDs or private instance details.",
      "Capture at 1280 by 900 in the light theme, then check that every label remains readable at a narrow viewport."
    ],
    "edits": [
      "Crop to retain the repository context, Project identity and state, Access status, and Join environment button; remove unrelated browser chrome only.",
      "Add at most three numbered markers: 1 for Project identity, 2 for Running state, and 3 for Not joined plus Join environment; explain all markers in the caption.",
      "Keep the real labels readable, do not rely on color alone, and verify the alt text against the final capture."
    ]
  }
}
```

A read-only repository grant, public visibility, prior membership, or Soda operator status does not grant execution access. If Join is denied, ask the repository owner for code-write access, then retry from the original Project. Do not create a second Project to work around a permission failure.

The Project keeps each member's original Linux login bound to their stable native identity. A native username change or repository transfer does not rename or remap the Linux account, file ownership, or installed SSH keys. Members have separate homes and checkouts, with shared tools, files, and explicitly assigned services in the common Project. See [Shared tools and files](../60-Develop/30-shared-tools-and-files.md) and [Project services](../60-Develop/40-project-services.md).

## Start and Stop the retained root

Choose **Start** on the existing Project when its retained environment is stopped. It reuses the same root, homes, installed tools, files, and service volumes. Start does not revive a terminal process, Factory conversation, or prior credential lease; open a new session after the Project is running.

Before choosing **Stop**, coordinate with other users and allow Factory withdrawal to settle. Stop affects every member: live terminals, SSH connections, and services are interrupted while persistent data remains. To resume later, explicitly Start the same Project and open a new terminal or execution. A successful Start restores the environment, not the interrupted processes.

## Prepare for Factory work

Human Project creation and joining do not prepare a Factory checkout. Before Factory work begins, accept the repository's exact setup inputs, have an administrator inspect and provide shared prerequisites, and establish readiness for unprivileged coder and reviewer roles. Changed inputs invalidate affected readiness; maintenance holds pause affected work until checks run again. Follow [Prepare a Project for Factory work](30-prepare-project.md).

Provider enrollment and Factory sponsorship are separate from Project setup. See [Provider connections](../30-Set-up/40-provider-connections.md) and [Factory authorization](../40-Factory/20-factory-authorization.md).
