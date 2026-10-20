# Complete public handbook tree

This is the implemented order of all **41 published pages in six sections**,
plus the unpublished README. Everyday use comes first, followed by configuration,
operations and installation. Each page keeps its existing semantic slug and
[reader contract](page-contracts.md); the [task list](cutover.md) retains its stable
page ID and remaining work.

```text
docs/public/
├── README.md
├── 10-Start-here/
│   ├── 10-index.md
│   ├── 20-product-model.md
│   ├── 30-spaces.md
│   ├── 40-first-connection.md
│   └── 50-provider-connections.md
├── 20-Factory/
│   ├── 10-software-factory.md
│   ├── 20-factory-feed.md
│   ├── 30-plan-new-work.md
│   ├── 40-issue-work-and-dependencies.md
│   ├── 50-review-and-merge.md
│   ├── 60-application-verification.md
│   ├── 70-knowledge-and-decisions.md
│   └── 80-factory-intervention.md
├── 30-Projects-and-development/
│   ├── 10-projects-and-workspaces.md
│   ├── 20-workspace-terminals.md
│   ├── 30-collaboration.md
│   ├── 40-connect-and-develop.md
│   ├── 50-shared-tools-and-files.md
│   ├── 60-project-services.md
│   └── 70-canlang-development.md
├── 40-Configure/
│   ├── 10-people-and-access.md
│   ├── 20-factory-authorization.md
│   ├── 30-prepare-project.md
│   ├── 40-ci-runners.md
│   ├── 50-private-networking-and-tls.md
│   └── 60-tailscale.md
├── 50-Operate/
│   ├── 10-troubleshooting.md
│   ├── 20-administration.md
│   ├── 30-cockpit.md
│   ├── 40-private-applications.md
│   ├── 50-backups-and-restoration.md
│   ├── 60-project-maintenance.md
│   ├── 70-updates-and-fallback.md
│   ├── 80-recovery.md
│   └── 90-data-safety-and-removal.md
└── 60-Install/
    ├── 10-deployment-options.md
    ├── 20-verify-downloads.md
    ├── 30-deploy-to-cloud.md
    ├── 40-install-on-premises.md
    ├── 50-docker.md
    └── 60-operator-setup.md
```

Section and page ordering comes from the filenames. The website maps `index` to
`/docs` and other slugs to `/docs/SLUG`; moving a page between folders does not
itself change its route. These are handbook URLs, not promises about application
UI routes.

There is no nested chapter directory, per-section README, generated content or
plan metadata inside the published tree. The optional `assets/` directory is
created only when an actual reviewed capture is added, following the existing
[capture guidance](../../design/screenshot-capture.md).
