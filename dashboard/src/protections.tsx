import { useEffect, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { Alert, Button, Form, FormGroup, Spinner, TextArea, TextInput } from "@patternfly/react-core";
import { APIError, request, type Session } from "./api";
import { useSession } from "./session";
import { repositoryPath } from "./repositories";
interface Branch { rule_name: string; enable_push?: boolean; require_signed_commits?: boolean; enable_status_check?: boolean; required_approvals?: number; block_on_rejected_reviews?: boolean; block_on_outdated_branch?: boolean; apply_to_admins?: boolean; dismiss_stale_approvals?: boolean; status_check_contexts?: string[] }
const flags = [["enable_push", "Allow native pushes"], ["require_signed_commits", "Require signed commits"], ["enable_status_check", "Require native status checks"], ["block_on_rejected_reviews", "Block rejected reviews"], ["block_on_outdated_branch", "Block outdated branch"], ["apply_to_admins", "Apply to native administrators"], ["dismiss_stale_approvals", "Dismiss stale approvals"]] as const;
export function BranchProtections({ session }: { session: Session }) {
  const { owner = "", repo = "" } = useParams(); const [query, setQuery] = useSearchParams(); const page = query.get("page") ?? "1"; const api = `/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/branch-protections`;
  const [items, setItems] = useState<Branch[] | null>(null); const [next, setNext] = useState<number | null>(null); const [selected, setSelected] = useState<Branch | null>(null); const [rule, setRule] = useState(""); const [changes, setChanges] = useState<Partial<Omit<Branch, "rule_name">>>({});
  const [contexts, setContexts] = useState("");
  const [attempt, setAttempt] = useState(0); const [error, setError] = useState(""); const [busy, setBusy] = useState(false);
  useEffect(() => {
    const controller = new AbortController(); setItems(null); setError("");
    request<{ items: Branch[]; next_page: number | null }>(`${api}?page=${encodeURIComponent(page)}`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) { setItems(value.items); setNext(value.next_page); } }).catch(failure => { if (controller.signal.aborted) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Branch protection unavailable."); }); return () => controller.abort();
  }, [api, page, attempt, session]);
  function select(value: Branch | null) { setSelected(value); setRule(value?.rule_name ?? ""); setContexts(value?.status_check_contexts?.join("\n") ?? ""); setChanges({}); }
  async function save() {
    if (busy) return; setBusy(true); setError("");
    try { await request(selected ? `${api}/${encodeURIComponent(selected.rule_name)}` : api, { method: selected ? "PATCH" : "POST", body: selected ? changes : { rule_name: rule, ...changes }, csrf: session.csrf_token }); if (useSession.getState().session === session) { select(null); setAttempt(value => value + 1); } }
    catch (failure) { if (useSession.getState().session !== session) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Protection change not confirmed."); } finally { setBusy(false); }
  }
  return <section aria-busy={busy}><h1>Native branch protection: {owner}/{repo}</h1><nav><Link to={`${repositoryPath(owner, repo)}/settings`}>Repository settings</Link><Link to={`${repositoryPath(owner, repo)}/tag-protections`}>Tag protection</Link></nav>{error && <Alert isInline variant="danger" title={error} />}{!items && !error && <Spinner aria-label="Loading branch protections" />}
    <ul>{items?.map(item => <li key={item.rule_name}>{item.rule_name} — required approvals {item.required_approvals ?? "not supplied"}; pushes {item.enable_push === undefined ? "not supplied" : item.enable_push ? "allowed" : "disabled"}<Button variant="link" isDisabled={busy} onClick={() => select(item)}>Edit rule {item.rule_name}</Button></li>)}</ul>
    <Button variant="secondary" isDisabled={busy || Number(page) <= 1} onClick={() => setQuery({ page: String(Number(page) - 1) })}>Previous page</Button><Button variant="secondary" isDisabled={busy || next === null} onClick={() => next !== null && setQuery({ page: String(next) })}>Next page</Button>
    <h2>{selected ? `Edit ${selected.rule_name}` : "Create native rule"}</h2>{selected && <Button variant="link" isDisabled={busy} onClick={() => select(null)}>New rule instead</Button>}<p>Only changed fields are patched; native settings and last-write rules apply. Advanced allowlist and file-pattern editing is available through the bounded API but its custom form remains source work. No protection is simulated in Soda and no environment operation occurs.</p>
    <Form onSubmit={event => { event.preventDefault(); void save(); }}><FormGroup label="Native branch rule name/pattern" fieldId="branch-rule"><TextInput id="branch-rule" value={rule} isDisabled={busy || Boolean(selected)} onChange={(_, value) => setRule(value)} /></FormGroup>
      {flags.map(([field, label]) => <label key={field}><input type="checkbox" checked={changes[field] ?? selected?.[field] ?? false} disabled={busy} onChange={event => { const value = event.currentTarget.checked; setChanges(current => ({ ...current, [field]: value })); }} /> {label}</label>)}
      <FormGroup label="Required native approvals" fieldId="required-approvals"><TextInput id="required-approvals" type="number" min={0} max={1000} value={changes.required_approvals ?? selected?.required_approvals ?? 0} isDisabled={busy} onChange={(_, value) => setChanges(current => ({ ...current, required_approvals: Number(value) }))} /></FormGroup>
      <FormGroup label="Required status contexts (one per line)" fieldId="required-contexts"><TextArea id="required-contexts" value={contexts} isDisabled={busy} onChange={(_, value) => { setContexts(value); setChanges(current => ({ ...current, status_check_contexts: value.split("\n").map(part => part.trim()).filter(Boolean) })); }} /></FormGroup><Button type="submit" isDisabled={busy || !rule || (Boolean(selected) && Object.keys(changes).length === 0)}>Save native branch rule</Button>
    </Form>
  </section>;
}
interface Tag { id: string; name_pattern: string; whitelist_usernames: string[]; whitelist_teams: string[] }
export function TagProtections({ session }: { session: Session }) {
  const { owner = "", repo = "" } = useParams(); const [query, setQuery] = useSearchParams(); const page = query.get("page") ?? "1"; const api = `/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/tag-protections`;
  const [items, setItems] = useState<Tag[] | null>(null); const [next, setNext] = useState<number | null>(null); const [selected, setSelected] = useState<Tag | null>(null); const [pattern, setPattern] = useState(""); const [users, setUsers] = useState(""); const [teams, setTeams] = useState("");
  const [attempt, setAttempt] = useState(0); const [error, setError] = useState(""); const [busy, setBusy] = useState(false);
  useEffect(() => {
    const controller = new AbortController(); setItems(null); setError("");
    request<{ items: Tag[]; next_page: number | null }>(`${api}?page=${encodeURIComponent(page)}`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) { setItems(value.items); setNext(value.next_page); } }).catch(failure => { if (controller.signal.aborted) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Tag protection unavailable."); }); return () => controller.abort();
  }, [api, page, attempt, session]);
  function select(value: Tag | null) { setSelected(value); setPattern(value?.name_pattern ?? ""); setUsers(value?.whitelist_usernames.join(", ") ?? ""); setTeams(value?.whitelist_teams.join(", ") ?? ""); }
  async function save() {
    if (busy) return; setBusy(true); setError("");
    try { await request(selected ? `${api}/${selected.id}` : api, { method: selected ? "PATCH" : "POST", body: { name_pattern: pattern, whitelist_usernames: users.split(",").map(value => value.trim()).filter(Boolean), whitelist_teams: teams.split(",").map(value => value.trim()).filter(Boolean) }, csrf: session.csrf_token }); if (useSession.getState().session === session) { select(null); setAttempt(value => value + 1); } }
    catch (failure) { if (useSession.getState().session !== session) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Tag protection change not confirmed."); } finally { setBusy(false); }
  }
  return <section aria-busy={busy}><h1>Native tag protection: {owner}/{repo}</h1><nav><Link to={`${repositoryPath(owner, repo)}/settings`}>Repository settings</Link><Link to={`${repositoryPath(owner, repo)}/branch-protections`}>Branch protection</Link></nav>{error && <Alert isInline variant="danger" title={error} />}{!items && !error && <Spinner aria-label="Loading tag protections" />}
    <ul>{items?.map(item => <li key={item.id}>{item.name_pattern} — users {item.whitelist_usernames.join(", ") || "none"}; teams {item.whitelist_teams.join(", ") || "none"}<Button variant="link" isDisabled={busy} onClick={() => select(item)}>Edit tag rule {item.id}</Button></li>)}</ul>
    <Button variant="secondary" isDisabled={busy || Number(page) <= 1} onClick={() => setQuery({ page: String(Number(page) - 1) })}>Previous page</Button><Button variant="secondary" isDisabled={busy || next === null} onClick={() => next !== null && setQuery({ page: String(next) })}>Next page</Button>
    <h2>{selected ? `Edit rule ${selected.id}` : "Create native tag rule"}</h2>{selected && <Button variant="link" isDisabled={busy} onClick={() => select(null)}>New rule instead</Button>}<p>Forgejo resolves allowed users/organization teams and enforces protection. Edits use native last-write behavior, not Soda permissions or Linux account synchronization.</p>
    <Form onSubmit={event => { event.preventDefault(); void save(); }}><FormGroup label="Native tag name pattern" fieldId="tag-pattern"><TextInput id="tag-pattern" value={pattern} isDisabled={busy} onChange={(_, value) => setPattern(value)} /></FormGroup><FormGroup label="Allowed native usernames (comma separated)" fieldId="tag-users"><TextInput id="tag-users" value={users} isDisabled={busy} onChange={(_, value) => setUsers(value)} /></FormGroup><FormGroup label="Allowed native organization teams (comma separated)" fieldId="tag-teams"><TextInput id="tag-teams" value={teams} isDisabled={busy} onChange={(_, value) => setTeams(value)} /></FormGroup><Button type="submit" isDisabled={busy || !pattern}>Save native tag rule</Button></Form>
  </section>;
}
