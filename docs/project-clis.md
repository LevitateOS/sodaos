# Forgejo and GitHub CLIs in projects — unbuilt source

The predecessor supplied Tea and GitHub CLI as developer tools. Here they belong in the **Rocky project image**, not the immutable host or a new Soda credential service. They are separate from the host's GitHub Actions runner client.

## Source and packaging

- **Tea 0.15.1:** retained `tea-source.toml`, source/archive/license checksums, license and fetch logic from `soda-os` commit `bc1d3e0dbec48dfaa6a20f9d0453ad3e4cdf353c`. `scripts/build-project-tools.py` uses upstream's own Makefile/version flags with static Go on matching-native Linux. The binary and license feed the project image; no Fedora RPM is copied into Rocky.
- **GitHub CLI 2.97.0:** keeps the predecessor's CLI version baseline but uses GitHub's official RPM repository inside Rocky, with GPG checking enabled. Both amd64 and arm64 release RPMs were present in inspected upstream release metadata. Actual repository availability, dependency closure and Rocky installation are still unverified.

`build-native.sh` invokes the Tea fetch/build before the project image build. The project Containerfile now uses the repository root as context and requires `ARTIFACT_DIR`; use the documented entrypoint rather than the previous standalone `project-os/` context. Native builder prerequisites include GNU make. Tea's version check executes only the newly built binary, never a login; provider behavior is a separate later test.

No CLI source archive, dependency, client binary or RPM was downloaded/installed during this source port. Only source/configuration and public upstream metadata were inspected/copied. New tools appear in newly built project images; existing persistent projects are not recreated or automatically upgraded to acquire them.

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
