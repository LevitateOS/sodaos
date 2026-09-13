# Forgejo and GitHub CLIs in projects

The predecessor supplied Tea and GitHub CLI as developer tools. Here they belong in the **Rocky project image**, not the immutable host or a new Soda credential service. These project tools are independent of Soda’s Forgejo-only CI runner support.

The planned Fedora/KDE variants inherit these Project OS tooling and personal-auth
contracts. Reuse the existing packaging/license owners and verify distro-specific
packaging; do not create separate GUI installers or copy one user's authenticated
CLI state. Desktop-launched tools must resolve the same account, home and shared
paths as the terminal. Current recipe/evidence below applies to Rocky; adding a
profile does not upgrade the tool pins or imply authenticated provider compatibility.
Unattended AI jobs use dedicated run credentials as defined in the
[AI plan](services-and-ai-plan.md), not a member's personal login.

## Source and packaging

- **Tea 0.16.0:** use the official upstream Linux amd64/arm64 binaries from [the release directory](https://dl.gitea.com/tea/0.16.0/). `project-os/locks/tea-binary.toml` records the release URLs and upstream SHA-256 digests. `scripts/fetch-tea.py` verifies the binary checksum, ELF architecture and retained upstream license, then stages the executable and license for the project image. No Tea source archive, Make invocation or Go compilation remains. This selects the newest published release found on 13 September 2026, following the owner's latest-version preference; the manifest identifies the exact downloaded bytes rather than requiring retention of an older release.
- **GitHub CLI 2.97.0:** keeps the predecessor's CLI version baseline. Native execution found that the rolling official RPM repository no longer indexes it. The image now selects the retained architecture-specific release RPM, imports the same official vendor key and enforces `localpkg_gpgcheck=1`; it does not upgrade the pin or disable signature checks. The downloaded amd64 RPM matched GitHub's published digest and passed RPM signature verification with that key (the importer warned about key expiry). Subsequent native x86_64 image builds completed this installation; arm64 metadata remains distinct from native arm64 proof.

Both `build-native.sh` and the complete host-image builder invoke the same Tea binary fetcher before building Project OS. Downloading and inspecting either architecture works locally without executing a Linux binary. The project Containerfile uses the repository root as context and requires `ARTIFACT_DIR`; use the documented build entrypoints. Existing native image metadata collection runs `tea --version` inside the resulting Project OS image. Personal login/provider behavior remains a separate check.

The previous Tea source fetcher, compiler wrapper, source lock and source-build tests were removed on 13 September 2026. The license is unchanged in upstream 0.16.0. Both actual Linux binaries downloaded through the replacement fetcher and passed SHA-256 and ELF architecture validation on the local Mac. Six fixture tests cover both architectures, checksum/license failures, incorrect architecture, network failure and preservation of existing output. The metadata collector tests passed (two unrelated Caddy integration tests were skipped). `go test ./internal/nativebuild ./tools/soda-host-image` passed with Go 1.26.7 and a real `/private/tmp` temporary parent; the initial macOS symlinked temp path caused two fixture-path failures. The host-image command compiled but has no package tests. Shell syntax and diff checks also passed. Native Linux execution and a new Project OS image build have not been performed for this replacement.

Historical authorized x86_64 builds downloaded and compiled Tea 0.15.1 before this replacement. Two concrete packaging defects were corrected: inherited Make GOFLAGS were exported in a form Go rejects, and Tea's unconditional ANSI version framing defeated the old version matcher. That corrected build then reached the pinned GitHub CLI repository-availability failure described above. Later builds completed the revised packaging: the retained `dad2945` x86_64 native build metadata records Tea 0.15.1 and GitHub CLI 2.97.0. This is historical image/tool evidence, not a current audit of every project or authenticated CLI compatibility. No personal CLI login/provider mutation is claimed. New tools appear in newly built images; existing persistent projects need the [Project OS same-root maintenance contract](project-os.md#deliver-required-additions-without-replacing-roots), not recreation or an assumed automatic image upgrade.

## Later personal authentication

After an authorized build/install, verify availability with `tea --version` and `gh --version` as the project-local user. Then authenticate **in a private interactive terminal** using your own account and the configured reachable service URL:

```sh
tea logins add
tea whoami
gh auth login --git-protocol ssh
gh auth status
```

For Tea, enter the deployment's Forgejo HTTPS origin, not the CLI's public Gitea default. Abort an unexpected hostname or certificate prompt. Follow Tea's interactive prompts and [native CLI guidance](https://docs.codeberg.org/git/clone-commit-via-cli/) and GitHub's [authentication documentation](https://cli.github.com/manual/gh_auth_login). These operations create real personal authentication state and can register Git keys if you choose that action; they are not read-only validation commands.

Browser sign-in, Soda sessions, incoming project SSH keys, Git transport credentials and these CLI sessions are distinct. Do not pass tokens in command arguments/shared logs, copy another person's credentials or place the CLI configuration in `~/shared`. Inspect native credential-file permissions; do not assume the CLI provides an encrypted host keyring inside the container. Trust the deployment's certificate authority through native project OS administration rather than disabling TLS checks.

Use each provider's native logout/revocation when needed. Soda does not propagate that lifecycle across systems. No CLI login or provider/API compatibility test has run here.
