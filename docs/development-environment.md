# Working inside a project

Connect using the project's displayed IP: `ssh alice@10.89.0.2` (example only). The operator must establish the deployment route to the project subnet; a private bridge address alone is not client connectivity.

- `~/shared` accesses `/srv/project/shared`, shared by project members through the native `soda-project` group.
- Clone ordinary repositories into your home, such as `~/repo-name`. Soda does not fork or synchronize these checkouts.
- Configure Git credentials in Forgejo or the external Git host. Soda's development-access key registration is a separate SSH authentication path; no private key upload or automatic agent forwarding is required.

## Shared installed tools

The shared mise data/install/shim tree is `/opt/mise`; its global version configuration is `/etc/mise/config.toml`. These native environment variables were derived from mise 2026.9.1's configuration documentation. Shared installations remain within one project filesystem; they are not copied between machines.

As the project administrator:

```sh
sudo -i
umask 022
mise use --global node@24
exit
```

Both Alice and Bob then resolve the same installed Node through `/opt/mise/shims`. The profile and SSH server set the environment for interactive and non-interactive sessions. Ordinary members may consume but not overwrite the root-owned shared toolchain. A repository can carry normal mise configuration. Additional missing versions in that configuration require installation through the appropriate native permissions, not a Soda-specific downloader.

Personal tools can be installed under personal paths through native commands (with explicit personal mise environment overrides if desired). There is no Soda-owned private-toolchain clone/selector. Changes to a repository, installed tool or service do not automatically promote or merge any runtime state.

Open a new shell after changing shell configuration. This is normal shell behavior, not a live environment propagation feature. Native resolution, executable permissions and two-user behavior remain unvalidated until the authored installed scenario is run later.
