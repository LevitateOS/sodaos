# Forgejo and GitHub CLIs in projects

The predecessor supplied Tea and GitHub CLI as developer tools. Here they belong in the **Rocky project image**, not the immutable host or a new Soda credential service. These project tools are independent of Soda’s Forgejo-only CI runner support.

The planned Fedora/KDE variants inherit these Project OS tooling and personal-auth
contracts. Reuse the existing source/license/build owners and verify distro-specific
packaging; do not create separate GUI installers or copy one user's authenticated
CLI state. Desktop-launched tools must resolve the same account, home and shared
paths as the terminal. Current recipe/evidence below applies to Rocky; adding a
profile does not upgrade the tool pins or imply authenticated provider compatibility.
Unattended AI jobs use dedicated run credentials as defined in the
[AI plan](services-and-ai-plan.md), not a member's personal login.

## Source and packaging

- **Tea 0.15.1:** retained `tea-source.toml`, source/archive/license checksums, license and fetch logic from `soda-os` commit `bc1d3e0dbec48dfaa6a20f9d0453ad3e4cdf353c`. `scripts/build-project-tools.py` uses upstream's own Makefile/version flags with static Go on matching-native Linux. The binary and license feed the project image; no Fedora RPM is copied into Rocky.
- **GitHub CLI 2.97.0:** keeps the predecessor's CLI version baseline. Native execution found that the rolling official RPM repository no longer indexes it. The image now selects the retained architecture-specific release RPM, imports the same official vendor key and enforces `localpkg_gpgcheck=1`; it does not upgrade the pin or disable signature checks. The downloaded amd64 RPM matched GitHub's published digest and passed RPM signature verification with that key (the importer warned about key expiry). Subsequent native x86_64 image builds completed this installation; arm64 metadata remains distinct from native arm64 proof.

`build-native.sh` invokes the Tea fetch/build before the project image build. The project Containerfile now uses the repository root as context and requires `ARTIFACT_DIR`; use the documented entrypoint rather than the previous standalone `project-os/` context. Native builder prerequisites include GNU make. Tea's version check executes only the newly built binary, never a login; provider behavior is a separate later test.

Subsequent authorized x86_64 builds downloaded and compiled Tea. Two concrete packaging defects were corrected: inherited Make GOFLAGS were exported in a form Go rejects, and Tea's unconditional ANSI version framing defeated the old version matcher. That corrected build then reached the pinned GitHub CLI repository-availability failure described above. Later builds completed the revised packaging: the retained `dad2945` x86_64 native build metadata records Tea 0.15.1 and GitHub CLI 2.97.0. This is historical image/tool evidence, not a current audit of every project or authenticated CLI compatibility. No personal CLI login/provider mutation is claimed. New tools appear in newly built images; existing persistent projects need the [Project OS same-root maintenance contract](project-os.md#deliver-required-additions-without-replacing-roots), not recreation or an assumed automatic image upgrade.

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
