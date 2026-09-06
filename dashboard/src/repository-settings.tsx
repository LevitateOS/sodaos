import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { Alert, Button, Form, FormGroup, Spinner, TextArea, TextInput } from "@patternfly/react-core";
import { APIError, request, type Session } from "./api";
import { useSession } from "./session";
import { repositoryPath } from "./repositories";
interface Settings { description: string; website: string; private: boolean; default_branch: string; has_issues: boolean; has_pull_requests: boolean; has_wiki: boolean; has_actions: boolean; has_releases: boolean; has_packages: boolean; allow_merge_commits: boolean; allow_squash_merge: boolean; allow_rebase: boolean; allow_rebase_explicit: boolean; allow_fast_forward_only_merge: boolean }
const switches = [["private", "Private repository"], ["has_issues", "Native issues"], ["has_pull_requests", "Native pull requests"], ["has_wiki", "Native wiki"], ["has_actions", "Native Actions"], ["has_releases", "Native releases"], ["has_packages", "Native packages"], ["allow_merge_commits", "Allow merge commits"], ["allow_squash_merge", "Allow squash merge"], ["allow_rebase", "Allow rebase"], ["allow_rebase_explicit", "Allow explicit rebase merge"], ["allow_fast_forward_only_merge", "Allow fast-forward-only merge"]] as const;
export function RepositorySettings({ session }: { session: Session }) {
  const { owner = "", repo = "" } = useParams(); const route = repositoryPath(owner, repo); const api = `/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/settings`;
  const [settings, setSettings] = useState<Settings | null>(null); const [changes, setChanges] = useState<Partial<Settings>>({}); const [error, setError] = useState(""); const [busy, setBusy] = useState(false);
  useEffect(() => {
    const controller = new AbortController(); setSettings(null); setChanges({}); setError("");
    request<Settings>(api, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) setSettings(value); }).catch(failure => { if (controller.signal.aborted) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Repository settings unavailable."); }); return () => controller.abort();
  }, [api, session]);
  async function save() {
    if (busy || Object.keys(changes).length === 0) return; setBusy(true); setError("");
    try { const value = await request<Settings>(api, { method: "PATCH", body: changes, csrf: session.csrf_token }); if (useSession.getState().session === session) { setSettings(value); setChanges({}); } }
    catch (failure) { if (useSession.getState().session !== session) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Settings not confirmed; inspect native state before retrying."); } finally { setBusy(false); }
  }
  return <section aria-busy={busy}><h1>Repository settings: {owner}/{repo}</h1><nav><Link to={route}>Repository</Link><Link to={`${route}/collaborators`}>Collaborators</Link><Link to={`${route}/deploy-keys`}>Deploy keys</Link><Link to={`${route}/hooks`}>Webhooks</Link><Link to={`${route}/branch-protections`}>Branch protection</Link><Link to={`${route}/tag-protections`}>Tag protection</Link></nav>
    <p>Forgejo enforces repository administration. Only changed fields are submitted, with native last-write behavior. Visibility and permission edits do not revoke existing Linux access, rename accounts, promote workloads or alter a project environment. Repository rename, transfer, archival and deletion are not exposed.</p>
    {error && <Alert isInline variant="danger" title={error} />}{!settings && !error && <Spinner aria-label="Loading repository settings" />}{settings && <Form onSubmit={event => { event.preventDefault(); void save(); }}>
      <FormGroup label="Repository description" fieldId="repository-description"><TextArea id="repository-description" value={changes.description ?? settings.description} isDisabled={busy} onChange={(_, value) => setChanges(current => ({ ...current, description: value }))} /></FormGroup>
      <FormGroup label="Repository website" fieldId="repository-website"><TextInput id="repository-website" value={changes.website ?? settings.website} isDisabled={busy} onChange={(_, value) => setChanges(current => ({ ...current, website: value }))} /></FormGroup>
      <FormGroup label="Default branch" fieldId="repository-default-branch"><TextInput id="repository-default-branch" value={changes.default_branch ?? settings.default_branch} isDisabled={busy} onChange={(_, value) => setChanges(current => ({ ...current, default_branch: value }))} /></FormGroup>
      {switches.map(([field, label]) => <label key={field}><input type="checkbox" checked={changes[field] ?? settings[field]} disabled={busy} onChange={event => { const value = event.currentTarget.checked; setChanges(current => ({ ...current, [field]: value })); }} /> {label}</label>)}
      <Button type="submit" isDisabled={busy || Object.keys(changes).length === 0}>Save changed native settings</Button>
    </Form>}
  </section>;
}
