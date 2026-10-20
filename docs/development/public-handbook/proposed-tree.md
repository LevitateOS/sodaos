# Complete target public tree

This is the selected replacement for the entire `docs/public/` tree. It contains
41 published pages and one unpublished README. Every page has a distinct
[contract](page-contracts.md); all current files have a
[cutover disposition](cutover.md#existing-file-disposition).

```text
docs/public/
├── README.md
├── 10-Start-here/
│   ├── 10-index.md
│   └── 20-product-model.md
├── 20-Install/
│   ├── 10-deployment-options.md
│   ├── 20-verify-downloads.md
│   ├── 30-install-on-premises.md
│   ├── 40-deploy-to-cloud.md
│   └── 50-docker.md
├── 30-Set-up/
│   ├── 10-operator-setup.md
│   ├── 20-private-networking-and-tls.md
│   ├── 30-people-and-access.md
│   ├── 40-provider-connections.md
│   ├── 50-first-connection.md
│   └── 60-tailscale.md
├── 40-Factory/
│   ├── 05-software-factory.md
│   ├── 10-plan-new-work.md
│   ├── 20-factory-authorization.md
│   ├── 30-issue-work-and-dependencies.md
│   ├── 40-ci-runners.md
│   ├── 50-review-and-merge.md
│   ├── 60-application-verification.md
│   ├── 70-factory-feed.md
│   ├── 80-knowledge-and-decisions.md
│   └── 90-factory-intervention.md
├── 50-Spaces-and-Projects/
│   ├── 10-spaces.md
│   ├── 20-projects-and-workspaces.md
│   ├── 30-prepare-project.md
│   └── 40-workspace-terminals.md
├── 60-Develop/
│   ├── 10-collaboration.md
│   ├── 20-connect-and-develop.md
│   ├── 30-shared-tools-and-files.md
│   ├── 40-project-services.md
│   └── 50-canlang-development.md
└── 70-Operate/
    ├── 10-administration.md
    ├── 20-cockpit.md
    ├── 30-project-maintenance.md
    ├── 40-private-applications.md
    ├── 50-updates-and-fallback.md
    ├── 60-backups-and-restoration.md
    ├── 70-data-safety-and-removal.md
    ├── 80-recovery.md
    └── 90-troubleshooting.md
```

Section and page ordering comes from the filenames. The website maps `index` to
`/docs` and other slugs to `/docs/SLUG`; moving a page between folders does not
itself change its route. These are handbook URLs, not promises about application
UI routes.

There is no nested chapter directory, per-section README, generated content or
plan metadata inside the published tree. The optional `assets/` directory is
created only when an actual reviewed capture is added, following the existing
[capture guidance](../../design/screenshot-capture.md).
