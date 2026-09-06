# React dashboard preview

Client-rendered React/PatternFly/Vite+/Zustand source. Go serves production files under `/app/`; the existing HTMX dashboard remains the default. Current connected features: real session discovery/local logout, Soda display-name preferences and development public-key registration/listing. Projects/People explicitly use legacy links; Forgejo-backed replacement views are still pending.

## Build boundary

The manifest reuses the current Cockpit baseline and adds React Router 7.18.3 after public metadata inspection. No dashboard dependencies have been installed/resolved, and no lockfile or generated assets are fabricated. In an explicitly authorized dependency phase, resolve and review the real `pnpm-lock.yaml` and package/license closure, then commit it. Frozen build entrypoints intentionally refuse a missing lockfile.

Later, with native build permission:

```sh
bash scripts/build-dashboard.sh x86_64
```

This builds only the dashboard frontend/binary/image into the established native artifact tree. It does not rebuild the project image, deploy, restart services or change VM state. Existing dashboard artifact outputs are refused; preserve the previous candidate and explicitly clear only approved generated outputs before another build. The full native build invokes the same entrypoint with `--payload-only`: this still builds the React assets and Go binary, but leaves image creation/export to the full build's native artifact packaging. That caller builds the dashboard image once and stages the same assets. Without the flag, the standalone dashboard-only build still produces its binary, assets and image without requiring the support pipeline.

The Go command's `--frontend-dir` defaults to `/usr/local/share/soda/dashboard`. A nonempty index, license payload, safe files and complete Vite manifest are validated **before** database opening/migration. Packaged frontend/backend bytes must be deployed together. See [API and migration notes](../docs/dashboard-api.md).

## Development and verification

`vp dev` binds loopback only. A standalone Vite server is **not** a configured authenticated development appliance: do not weaken cookies/CSRF/TLS or add permissive CORS to make its different origin work. Use the compiled preview served by Go behind the approved HTTPS endpoint for the first real login journey. A separately configured trusted-HTTPS development proxy/Fast Refresh arrangement remains U02 follow-up.

Go API/migration/asset tests and frontend fetch/store/form tests are authored, not executed. The check-native entrypoint includes the new TypeScript/UI checks once dependencies and staging exist. Browser/private-network/native proof remains separately authorized; source tests are not installed evidence.

Production has no Node service, browser token storage, SSR, Tailwind or TanStack. The build reuses only the pure license collector/vendor notices from Cockpit—not its privileged runtime bridge. Canonical art/palette are imported from `assets/`.
