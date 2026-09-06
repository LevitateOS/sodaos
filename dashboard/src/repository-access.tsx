import { useEffect, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { Alert, Button, Form, FormGroup, FormSelect, FormSelectOption, Spinner, TextArea, TextInput } from "@patternfly/react-core";
import { APIError, request, type Session } from "./api";
import { useSession } from "./session";
import { repositoryPath } from "./repositories";
interface Person { id: string; login: string }
function CollaboratorPermission({ api, login, session }: { api: string; login: string; session: Session }) {
  const [value, setValue] = useState<{ permission: string; role_name: string; user: Person } | null>(null); const [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController();
    request<{ permission: string; role_name: string; user: Person }>(`${api}/${encodeURIComponent(login)}`, { signal: controller.signal }).then(result => { if (!controller.signal.aborted) setValue(result); }).catch(failure => { if (controller.signal.aborted) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Permission unavailable."); }); return () => controller.abort();
  }, [api, login, session]);
  return <section aria-label="Native collaborator permission">{error && <Alert isInline variant="warning" title={error} />}{value && <p>{value.user.login}: native permission {value.permission}, role {value.role_name || "not supplied"}.</p>}</section>;
}
export function Collaborators({ session }: { session: Session }) {
  const { owner = "", repo = "" } = useParams(); const [query, setQuery] = useSearchParams(); const page = query.get("page") ?? "1"; const api = `/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/collaborators`;
  const [items, setItems] = useState<Person[] | null>(null); const [next, setNext] = useState<number | null>(null); const [selected, setSelected] = useState(""); const [login, setLogin] = useState(""); const [permission, setPermission] = useState("read"); const [confirmed, setConfirmed] = useState(false); const [attempt, setAttempt] = useState(0); const [busy, setBusy] = useState(false); const [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController(); setItems(null); setSelected(""); setError("");
    request<{ items: Person[]; next_page: number | null }>(`${api}?page=${encodeURIComponent(page)}`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) { setItems(value.items); setNext(value.next_page); } }).catch(failure => { if (controller.signal.aborted) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Collaborators unavailable."); }); return () => controller.abort();
  }, [api, page, attempt, session]);
  async function save(remove: boolean) {
    if (busy || !login || (remove && !confirmed)) return; setBusy(true); setError("");
    try {
      await request(`${api}/${encodeURIComponent(login)}`, { method: remove ? "DELETE" : "PUT", body: remove ? {} : { permission }, csrf: session.csrf_token });
      if (useSession.getState().session === session) { setConfirmed(false); setAttempt(value => value + 1); }
    } catch (failure) { if (useSession.getState().session !== session) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Collaborator change not confirmed."); } finally { setBusy(false); }
  }
  return <section aria-busy={busy}><h1>Native collaborators: {owner}/{repo}</h1><Link to={`${repositoryPath(owner, repo)}/settings`}>Repository settings</Link><p>These are Forgejo repository permissions, not Soda environment memberships. Removing a direct collaborator may leave inherited native team access and does not revoke Linux access.</p>
    {error && <Alert isInline variant="danger" title={error} />}{!items && !error && <Spinner aria-label="Loading collaborators" />}<ul>{items?.map(person => <li key={person.id}>{person.login}<Button variant="link" isDisabled={busy} onClick={() => setSelected(person.login)}>Inspect {person.login} permission</Button></li>)}</ul>{selected && <CollaboratorPermission key={selected} api={api} login={selected} session={session} />}
    <Button variant="secondary" isDisabled={busy || Number(page) <= 1} onClick={() => setQuery({ page: String(Number(page) - 1) })}>Previous page</Button><Button variant="secondary" isDisabled={busy || next === null} onClick={() => next !== null && setQuery({ page: String(next) })}>Next page</Button>
    <Form onSubmit={event => { event.preventDefault(); void save(false); }}><FormGroup label="Native collaborator username" fieldId="collaborator-login"><TextInput id="collaborator-login" value={login} isDisabled={busy} onChange={(_, value) => { setLogin(value); setConfirmed(false); }} /></FormGroup><FormGroup label="Native repository permission" fieldId="collaborator-permission"><FormSelect id="collaborator-permission" value={permission} isDisabled={busy} onChange={(_, value) => setPermission(value)}>{["read", "write", "admin"].map(value => <FormSelectOption key={value} value={value} label={value} />)}</FormSelect></FormGroup><Button type="submit" isDisabled={busy || !login}>Set native collaborator permission</Button><label><input type="checkbox" checked={confirmed} disabled={busy} onChange={event => setConfirmed(event.currentTarget.checked)} /> Remove the direct native collaborator entry for {login || "the entered username"}</label><Button variant="danger" isDisabled={busy || !login || !confirmed} onClick={() => void save(true)}>Remove direct collaborator</Button></Form>
  </section>;
}
interface DeployKey { id: string; title: string; key: string; fingerprint: string; read_only: boolean }
export function DeployKeys({ session }: { session: Session }) {
  const { owner = "", repo = "" } = useParams(); const [query, setQuery] = useSearchParams(); const page = query.get("page") ?? "1"; const api = `/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/deploy-keys`;
  const [items, setItems] = useState<DeployKey[] | null>(null); const [next, setNext] = useState<number | null>(null); const [title, setTitle] = useState(""); const [key, setKey] = useState(""); const [readOnly, setReadOnly] = useState(true); const [removing, setRemoving] = useState<DeployKey | null>(null); const [attempt, setAttempt] = useState(0); const [busy, setBusy] = useState(false); const [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController(); setItems(null); setError("");
    request<{ items: DeployKey[]; next_page: number | null }>(`${api}?page=${encodeURIComponent(page)}`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) { setItems(value.items); setNext(value.next_page); } }).catch(failure => { if (controller.signal.aborted) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Deploy keys unavailable."); }); return () => controller.abort();
  }, [api, page, attempt, session]);
  async function save(remove: boolean) {
    if (busy || (remove && !removing)) return; setBusy(true); setError("");
    try {
      await request(remove ? `${api}/${removing!.id}` : api, { method: remove ? "DELETE" : "POST", body: remove ? {} : { title, key, read_only: readOnly }, csrf: session.csrf_token });
      if (useSession.getState().session === session) { if (remove) setRemoving(null); else { setTitle(""); setKey(""); } setAttempt(value => value + 1); }
    } catch (failure) { if (useSession.getState().session !== session) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Deploy key change not confirmed."); } finally { setBusy(false); }
  }
  return <section aria-busy={busy}><h1>Native deploy keys: {owner}/{repo}</h1><Link to={`${repositoryPath(owner, repo)}/settings`}>Repository settings</Link><p>Repository Git deploy keys are separate from personal Forgejo Git keys and Soda development SSH keys. No project account/key provisioning or revocation is performed here.</p>
    {error && <Alert isInline variant="danger" title={error} />}{!items && !error && <Spinner aria-label="Loading deploy keys" />}<ul>{items?.map(item => <li key={item.id}>{item.title} — {item.read_only ? "read only" : "write enabled"} — {item.fingerprint}<details><summary>Public key</summary><code>{item.key}</code></details><Button variant="link" isDisabled={busy} onClick={() => setRemoving(item)}>Remove deploy key {item.title}</Button></li>)}</ul>
    {removing && <Alert isInline variant="warning" title={`Remove native deploy key ${removing.title} (${removing.fingerprint})?`}><Button variant="danger" isDisabled={busy} onClick={() => void save(true)}>Confirm deploy-key removal</Button><Button variant="link" isDisabled={busy} onClick={() => setRemoving(null)}>Cancel removal</Button></Alert>}
    <Button variant="secondary" isDisabled={busy || Number(page) <= 1} onClick={() => setQuery({ page: String(Number(page) - 1) })}>Previous page</Button><Button variant="secondary" isDisabled={busy || next === null} onClick={() => next !== null && setQuery({ page: String(next) })}>Next page</Button>
    <Form onSubmit={event => { event.preventDefault(); void save(false); }}><FormGroup label="Deploy key title" fieldId="deploy-title"><TextInput id="deploy-title" value={title} isDisabled={busy} onChange={(_, value) => setTitle(value)} /></FormGroup><FormGroup label="Public SSH key — never a private key" fieldId="deploy-key"><TextArea id="deploy-key" value={key} isDisabled={busy} rows={4} onChange={(_, value) => setKey(value)} /></FormGroup><label><input type="checkbox" checked={readOnly} disabled={busy} onChange={event => setReadOnly(event.currentTarget.checked)} /> Read-only Git access</label><Button type="submit" isDisabled={busy || !title || !key}>Add native deploy key</Button></Form>
  </section>;
}
