# Operator bootstrap

Source baseline: Forgejo 15.0.7. Soda uses OAuth2 and `/api/v1/user`, not an assumed
OIDC identity-token flow.

Media continuation: [Installation media](media.md).
Install/activate context: [Installation](installation.md).

1. Provision operator host access and start the Forgejo unit. Its web listener
   initially binds host loopback. Do not expose an unfinished installer through the
   public reverse proxy.
2. Use SSH forwarding (`ssh -L 33000:127.0.0.1:3000 root@HOST`) and open
   `http://localhost:33000` to complete Forgejo's native installation and create its
   administrator. Activation later applies the private HTTPS origin and native SSH
   clone port 2222.
3. Create an operator access token with `write:user` (includes `read:user`). Setup
   reads `/api/v1/user` and creates `/api/v1/user/applications/oauth2`. The token must
   belong to a Forgejo site administrator for Soda's bootstrap eligibility check.
   Supply it through a mode-0600 file; never expose it to developers.
4. Run `/usr/local/sbin/soda-setup --forgejo-url https://FORGEJO --token-file /root/forgejo-token`.
   Setup records `operator_id`, creates the OAuth client and retains the OAuth secret
   and grant-encryption key. It refuses to overwrite existing configuration.
5. Run `/usr/local/sbin/soda-activate` with the explicit private bind address and
   either `--local-tls` for that IP or an existing certificate/key. Local TLS requires
   explicit trust of the appliance's public root certificate on intended clients.

Use absolute setup/activation paths: CoreOS root SSH PATH may omit `/usr/local/sbin`.

Developers use native Forgejo account creation. Soda's API/OAuth service and Spaces
share the configured Forgejo HTTPS origin under `/-/soda/`. Git authentication uses
native Forgejo SSH keys or HTTPS tokens, independently of Soda development-access
keys.

Existing installations use [credential maintenance](../reference/credentials.md),
not rerunning setup. Console guidance: [Operator console welcome](../design/console-welcome.md).
