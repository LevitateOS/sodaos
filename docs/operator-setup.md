# Operator bootstrap (execution later)

Source baseline: Forgejo 15.0.7. API shapes were inspected in its upstream `templates/swagger/v1_json.tmpl`; authorization-code exchange and S256 PKCE are implemented in `routers/web/auth/oauth.go`. Soda uses OAuth2 and `/api/v1/user`, not an assumed OIDC identity-token flow.

1. On the eventual authorized native target, provision the operator's host access and start the Forgejo unit. Its web listener initially binds host loopback. Do not expose the unfinished installer through the public reverse proxy.
2. Use operator SSH forwarding (`ssh -L 3000:127.0.0.1:3000 root@HOST`) to complete Forgejo's native installation and create its administrator. Configure the eventual HTTPS Forgejo origin and native SSH clone port 2222. The first installer is operator work, not developer onboarding.
3. Through Forgejo's native settings create an operator access token with the required `read:user`, `write:user` (OAuth application creation), `write:admin` (people creation), and `read:repository` permissions. It is a server credential; never expose it to developers. Store it in a mode-0600 file on the appliance.
4. Run `soda-setup --public-url https://SODA --forgejo-url https://FORGEJO --token-file /root/forgejo-token`. This calls the native API, records the operator's stable provider ID and creates the OAuth client. It refuses to overwrite existing configuration. Inspect native Forgejo applications before retrying a partially completed setup; no reconciliation service is provided.
5. Apply the native service/configuration ownership described in the installation guide, configure trusted HTTPS for both browser origins, then enable the dashboard and proxy. The dashboard service is not root. Root/operator Cockpit authentication remains separate from Forgejo.

Developers are created by the operator's Soda dashboard. Their initial password is sent only to Forgejo; Soda does not store it. They change it through native Forgejo onboarding. Git authentication uses native Forgejo SSH keys or HTTPS tokens, independently of Soda development-access public keys.

Native API, TLS, container and login behavior has not been built or exercised. No live registration is performed by editing these sources.
