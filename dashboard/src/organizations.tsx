import { useEffect, useRef, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { Alert, Button, Form, FormGroup, FormSelect, FormSelectOption, Spinner, TextArea, TextInput } from "@patternfly/react-core";
import { APIError, request, type Session } from "./api";
import { useSession } from "./session";
import { OrganizationTeams } from "./teams";
export interface Organization { id: string; name: string; full_name: string; description: string; email: string; website: string; location: string; visibility: string; repo_admin_change_team_access: boolean }
export function Organizations({ session }: { session: Session }) {
  const [query, setQuery] = useSearchParams(); const page = query.get("page") ?? "1"; const mine = query.get("mine") ?? "false"; const [items, setItems] = useState<Organization[] | null>(null); const [next, setNext] = useState<number | null>(null); const [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController(); setItems(null); setError("");
    request<{ items: Organization[]; next_page: number | null }>(`/api/forgejo/organizations?${new URLSearchParams({ page, mine })}`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) { setItems(value.items); setNext(value.next_page); } }).catch(failure => { if (controller.signal.aborted) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Organizations unavailable."); }); return () => controller.abort();
  }, [page, mine, session]);
  return <section><h1>Native organizations</h1><Link to="/organizations/new">Create organization</Link><p>Forgejo owns organization membership and permissions. Organization repository access does not confer Soda operator authority or support organization-owned development environments.</p><label><input type="checkbox" checked={mine === "true"} onChange={event => setQuery({ mine: String(event.currentTarget.checked) })} /> My native organizations only</label>
    {error && <Alert isInline variant="danger" title={error} />}{!items && !error && <Spinner aria-label="Loading organizations" />}<ul>{items?.map(org => <li key={org.id}><Link to={`/organizations/${encodeURIComponent(org.name)}`}>{org.full_name || org.name}</Link> — {org.visibility}</li>)}</ul>
    <Button variant="secondary" isDisabled={Number(page) <= 1} onClick={() => setQuery({ mine, page: String(Number(page) - 1) })}>Previous page</Button><Button variant="secondary" isDisabled={next === null} onClick={() => next !== null && setQuery({ mine, page: String(next) })}>Next page</Button>
  </section>;
}
export function NewOrganization({ session }: { session: Session }) {
  const navigate = useNavigate(); const active = useRef(true); useEffect(() => { active.current = true; return () => { active.current = false; }; }, []);
  const [name, setName] = useState(""); const [fullName, setFullName] = useState(""); const [description, setDescription] = useState(""); const [visibility, setVisibility] = useState("public"); const [busy, setBusy] = useState(false); const [error, setError] = useState("");
  async function create() {
    if (busy) return; setBusy(true); setError("");
    try { const org = await request<Organization>("/api/forgejo/organizations", { method: "POST", body: { username: name, full_name: fullName, description, visibility }, csrf: session.csrf_token }); if (active.current && useSession.getState().session === session) navigate(`/organizations/${encodeURIComponent(org.name)}`); }
    catch (failure) { if (!active.current || useSession.getState().session !== session) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Creation not confirmed; inspect native organizations before retrying."); } finally { setBusy(false); }
  }
  return <section aria-busy={busy}><h1>Create native organization</h1><Link to="/organizations">Organizations</Link>{error && <Alert isInline variant="danger" title={error} />}<Form onSubmit={event => { event.preventDefault(); void create(); }}>
    <FormGroup label="Native organization name" fieldId="organization-name"><TextInput id="organization-name" value={name} isDisabled={busy} onChange={(_, value) => setName(value)} /></FormGroup><FormGroup label="Organization full name" fieldId="organization-full-name"><TextInput id="organization-full-name" value={fullName} isDisabled={busy} onChange={(_, value) => setFullName(value)} /></FormGroup><FormGroup label="Organization description" fieldId="organization-description"><TextArea id="organization-description" value={description} isDisabled={busy} onChange={(_, value) => setDescription(value)} /></FormGroup><FormGroup label="Native visibility" fieldId="organization-visibility"><FormSelect id="organization-visibility" value={visibility} isDisabled={busy} onChange={(_, value) => setVisibility(value)}>{["public", "limited", "private"].map(value => <FormSelectOption key={value} value={value} label={value} />)}</FormSelect></FormGroup><Button type="submit" isDisabled={busy || !name}>Create organization in Forgejo</Button>
  </Form><p>No development environment, host account or runner resource is created.</p></section>;
}
function OrganizationMembers({ session, org }: { session: Session; org: string }) {
  const [query, setQuery] = useSearchParams(); const page = query.get("page") ?? "1"; const [items, setItems] = useState<{ id: string; login: string }[] | null>(null); const [next, setNext] = useState<number | null>(null); const [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController(); setItems(null); setError("");
    request<{ items: { id: string; login: string }[]; next_page: number | null }>(`/api/forgejo/organizations/${encodeURIComponent(org)}/members?page=${encodeURIComponent(page)}`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) { setItems(value.items); setNext(value.next_page); } }).catch(failure => { if (controller.signal.aborted) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Native members unavailable."); }); return () => controller.abort();
  }, [org, page, session]);
  return <section><h2>Native organization members</h2>{error && <Alert isInline variant="warning" title={error} />}{!items && !error && <Spinner aria-label="Loading organization members" />}<ul>{items?.map(user => <li key={user.id}>{user.login}</li>)}</ul><p>Membership is managed through native teams; it is not a Soda authorization table or a Linux group inventory.</p><Button variant="secondary" isDisabled={Number(page) <= 1} onClick={() => setQuery({ tab: "members", page: String(Number(page) - 1) })}>Previous page</Button><Button variant="secondary" isDisabled={next === null} onClick={() => next !== null && setQuery({ tab: "members", page: String(next) })}>Next page</Button></section>;
}
function OrganizationSettings({ session, org, onChanged }: { session: Session; org: Organization; onChanged: (org: Organization) => void }) {
  const [changes, setChanges] = useState<Partial<Omit<Organization, "id" | "name">>>({}); const [busy, setBusy] = useState(false); const [error, setError] = useState("");
  async function save() {
    if (busy) return; setBusy(true); setError("");
    try { const value = await request<Organization>(`/api/forgejo/organizations/${encodeURIComponent(org.name)}`, { method: "PATCH", body: changes, csrf: session.csrf_token }); if (useSession.getState().session === session) { setChanges({}); onChanged(value); } }
    catch (failure) { if (useSession.getState().session !== session) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Organization settings not confirmed."); } finally { setBusy(false); }
  }
  return <section aria-busy={busy}><h2>Native organization profile/settings</h2>{error && <Alert isInline variant="danger" title={error} />}<p>The API adapter retains omitted profile strings because pinned native PATCH otherwise clears them. Native last-write rules and organization-owner permissions apply. Rename, delete and account remapping are not exposed.</p><Form onSubmit={event => { event.preventDefault(); void save(); }}>
    {[["full_name", "Full name"], ["description", "Description"], ["email", "Native contact email"], ["website", "Website"], ["location", "Location"]].map(([field, label]) => {
      const key = field as "full_name" | "description" | "email" | "website" | "location";
      return <FormGroup key={key} label={label} fieldId={`org-setting-${key}`}><TextInput id={`org-setting-${key}`} value={changes[key] ?? org[key]} isDisabled={busy} onChange={(_, value) => setChanges(current => ({ ...current, [key]: value }))} /></FormGroup>;
    })}
    <FormGroup label="Native visibility" fieldId="org-setting-visibility"><FormSelect id="org-setting-visibility" value={changes.visibility ?? org.visibility} isDisabled={busy} onChange={(_, value) => setChanges(current => ({ ...current, visibility: value }))}>{["public", "limited", "private"].map(value => <FormSelectOption key={value} value={value} label={value} />)}</FormSelect></FormGroup>
    <label><input type="checkbox" checked={changes.repo_admin_change_team_access ?? org.repo_admin_change_team_access} disabled={busy} onChange={event => { const value = event.currentTarget.checked; setChanges(current => ({ ...current, repo_admin_change_team_access: value })); }} /> Let native repository administrators change team access under Forgejo's rules</label><Button type="submit" isDisabled={busy || Object.keys(changes).length === 0}>Save changed native organization fields</Button>
  </Form></section>;
}
export function OrganizationDetail({ session }: { session: Session }) {
  const { org = "" } = useParams(); const [query, setQuery] = useSearchParams(); const tab = query.get("tab") ?? "teams"; const [value, setValue] = useState<Organization | null>(null); const [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController(); setValue(null); setError("");
    request<Organization>(`/api/forgejo/organizations/${encodeURIComponent(org)}`, { signal: controller.signal }).then(result => { if (!controller.signal.aborted) setValue(result); }).catch(failure => { if (controller.signal.aborted) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Native organization unavailable."); }); return () => controller.abort();
  }, [org, session]);
  return <section><h1>Organization: {value?.full_name || org}</h1><Link to="/organizations">Organizations</Link>{error && <Alert isInline variant="danger" title={error} />}{!value && !error && <Spinner aria-label="Loading organization" />}{value && <><p>{value.description}</p><p>{value.visibility} — {value.location}</p><Link to={`/organizations/${encodeURIComponent(org)}/actions/configuration`}>Actions secrets and variables</Link><nav aria-label="Organization tabs">{["teams", "members", "settings"].map(value => <Button key={value} variant={tab === value ? "primary" : "secondary"} onClick={() => setQuery({ tab: value })}>{value}</Button>)}</nav>{tab === "teams" && <OrganizationTeams session={session} org={org} />}{tab === "members" && <OrganizationMembers session={session} org={org} />}{tab === "settings" && <OrganizationSettings session={session} org={value} onChanged={setValue} />}<p>Organization-owned repositories remain native Forgejo resources; creating their Soda development environments is unsupported under the current human-owner rule.</p></>}</section>;
}
