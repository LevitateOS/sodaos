# Operator bootstrap

Source baseline: Forgejo 15.0.7. API shapes were inspected in its upstream `templates/swagger/v1_json.tmpl`; authorization-code exchange and S256 PKCE are implemented in `routers/web/auth/oauth.go`. Soda uses OAuth2 and `/api/v1/user`, not an assumed OIDC identity-token flow.

The replacement [manual installer](coreos-installer.md) guides this same bootstrap
with `soda-install configure`, run from the laptop's SSH terminal after local key
enrollment. Its source uses the selected private IP and Caddy local HTTPS, with
explicit client certificate trust. No domain ownership is required. Rebuilt media
and the full native journey remain pending; the delivered ISO has the old flow.

1. On the eventual authorized native target, provision the operator's host access and start the Forgejo unit. Its web listener initially binds host loopback. Do not expose the unfinished installer through the public reverse proxy.
2. Use operator SSH forwarding (`ssh -L 33000:127.0.0.1:3000 root@HOST`) and open `http://localhost:33000` to complete Forgejo's native installation and create its administrator. Use the reachable localhost browser URL during bootstrap; activation applies the eventual private HTTPS origin and native SSH clone port 2222. The first installer is operator work, not developer onboarding.
3. Through Forgejo's native settings create an operator access token with `write:user` (which includes `read:user`). Setup only reads `/api/v1/user` and creates `/api/v1/user/applications/oauth2`; neither requires admin or repository token scopes. The token must still belong to a Forgejo site administrator to satisfy Soda's existing bootstrap eligibility check. Never expose it to developers; supply it through a mode-0600 file on the appliance.
4. Run `/usr/local/sbin/soda-setup --forgejo-url https://FORGEJO --token-file /root/forgejo-token`. This calls the native API, records the operator's stable provider ID and creates the OAuth client. New setup retains only the OAuth secret and grant-encryption key, not a bootstrap-token copy/reference. The supplied token file is unchanged. It refuses to overwrite existing configuration. Inspect native Forgejo applications before retrying a partially completed setup; no reconciliation service is provided.
5. Run `/usr/local/sbin/soda-activate` with the explicit private bind address and either `--local-tls` for that IP or an existing certificate/key, as described in [installation](installation.md). Local TLS requires explicit trust of the appliance's public root certificate on intended clients. Activation applies native ownership and starts the dashboard/proxy. The dashboard service is not root. Root/operator Cockpit authentication remains separate from Forgejo.

The scope guidance follows selected Forgejo 15.0.7's `routers/api/v1/api.go` user
route group and `models/auth/access_token_scope.go` read/write inclusion, not the
misleading historical `admin:application` route comment. Local synthetic API checks
exercise Soda's request contract; actual scoped-token/provider setup remains native
validation. Existing installations require [separate credential maintenance](dashboard-credentials.md#retired-bootstrap-token--existing-install-maintenance),
not rerunning setup/activation. Revocation/deletion is a separate explicit operator
action, never an automatic migration.

Developers use native Forgejo account creation/onboarding; Soda has no account/password administration frontend. Soda's API/OAuth service and Sodaspaces use the same configured Forgejo HTTPS origin under `/-/soda/`. Git authentication uses native Forgejo SSH keys or HTTPS tokens, independently of Soda development-access public keys.

The merged source adds an interactive root [operator console welcome](console-welcome.md): observed interfaces, loopback Cockpit access and configured browser origins. It is guidance, not enrollment or a reachability check; its native deployment has not been validated.

The local x86_64 VM has completed this bootstrap and now serves the dashboard, Forgejo and Cockpit; see [local testing](local-testing.md). A real operator browser login through Forgejo OAuth, authenticated dashboard navigation and sign-out were verified with the test CA trusted. The full developer/project journey remains unverified. Use the absolute setup/activation command paths above: CoreOS's root SSH PATH omits `/usr/local/sbin`. No live registration is performed by editing these sources.
