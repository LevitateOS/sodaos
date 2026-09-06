import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { Alert, Button, Form, FormGroup, Spinner, TextArea, TextInput } from "@patternfly/react-core";
import { APIError, request, type Session } from "./api";
import { useSession } from "./session";

interface NativeUser { id: string; login: string; full_name: string; is_admin: boolean }
interface Settings { full_name: string; description: string; location: string; pronouns: string }
interface GitKey { id: string; title: string; key: string; fingerprint: string }
export function ForgejoAccount({ session }: { session: Session }) {
  const [user, setUser] = useState<NativeUser | null>(null); const [settings, setSettings] = useState<Settings | null>(null);
  const [keys, setKeys] = useState<GitKey[] | null>(null); const [keyPage, setKeyPage] = useState(1); const [next, setNext] = useState<number | null>(null);
  const [title, setTitle] = useState(""); const [key, setKey] = useState("");
  const [error, setError] = useState(""); const [notice, setNotice] = useState(""); const [busy, setBusy] = useState(false); const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    const controller = new AbortController(); setError(""); setKeys(null);
    Promise.all([
      request<NativeUser>("/api/forgejo/me", { signal: controller.signal }),
      request<Settings>("/api/forgejo/me/settings", { signal: controller.signal }),
      request<{ items: GitKey[]; next_page: number | null }>(`/api/forgejo/me/git-keys?page=${keyPage}`, { signal: controller.signal }),
    ]).then(([identity, profile, list]) => { if (!controller.signal.aborted) { setUser(identity); setSettings(profile); setKeys(list.items); setNext(list.next_page); } }).catch(failure => {
      if (controller.signal.aborted) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Forgejo account unavailable.");
    });
    return () => controller.abort();
  }, [session, keyPage, attempt]);
  async function save(kind: "settings" | "key") {
    if (busy) return; setBusy(true); setError(""); setNotice("");
    try {
      if (kind === "settings") await request("/api/forgejo/me/settings", { method: "PATCH", csrf: session.csrf_token, body: settings });
      else await request("/api/forgejo/me/git-keys", { method: "POST", csrf: session.csrf_token, body: { title, key } });
      if (useSession.getState().session !== session) return;
      setKey(""); setTitle(""); setNotice(kind === "settings" ? "Forgejo profile saved. Soda preferences were not changed." : "Forgejo Git key registered. No project account was changed."); setAttempt(value => value + 1);
    } catch (failure) {
      if (useSession.getState().session !== session) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Operation not confirmed. Inspect native state before retrying.");
    } finally { setBusy(false); }
  }
  return <section aria-busy={busy}><h1>Forgejo account</h1><p>Forgejo owns these account fields and Git keys. <Link to="/profile">Soda preferences and development-access keys</Link> are separate.</p>
    <p><a href={`${session.forgejo_url}/user/settings/security`}>Native account security</a> · <a href={`${session.forgejo_url}/user/settings/applications`}>Native applications and consent</a></p>
    {error && <Alert isInline variant="danger" title={error} />}{notice && <Alert isInline variant="success" title={notice} />}
    {!user && !error && <Spinner aria-label="Loading Forgejo account" />}
    {user?.is_admin && <p><Link to="/administration/people">Forgejo People administration</Link> — uses your current upstream administrator authority, not Soda operator or host-root rights.</p>}
    {settings && <Form onSubmit={event => { event.preventDefault(); void save("settings"); }}>
      {([['full_name', 'Full name'], ['description', 'Description'], ['location', 'Location'], ['pronouns', 'Pronouns']] as const).map(([field, label]) => <FormGroup key={field} label={label} fieldId={`native-${field}`}><TextInput id={`native-${field}`} value={settings[field]} isDisabled={busy} onChange={(_, value) => setSettings({ ...settings, [field]: value })} /></FormGroup>)}
      <Button type="submit" isDisabled={busy}>Save Forgejo profile</Button>
    </Form>}
    <h2>Forgejo Git SSH keys</h2><ul>{keys?.map(item => <li key={item.id}>{item.title} <code>{item.fingerprint}</code></li>)}</ul>{keys?.length === 0 && <p>No keys on this page.</p>}
    <Button variant="secondary" isDisabled={busy || keyPage <= 1} onClick={() => setKeyPage(value => value - 1)}>Previous key page</Button><Button variant="secondary" isDisabled={busy || next === null} onClick={() => next !== null && setKeyPage(next)}>Next key page</Button>
    <Form onSubmit={event => { event.preventDefault(); void save("key"); }}>
      <FormGroup label="Git key title" fieldId="git-key-title" isRequired><TextInput id="git-key-title" value={title} isDisabled={busy} isRequired onChange={(_, value) => setTitle(value)} /></FormGroup>
      <FormGroup label="Public Git SSH key — never a private key" fieldId="git-key" isRequired><TextArea id="git-key" value={key} isDisabled={busy} isRequired spellCheck={false} onChange={(_, value) => setKey(value)} /></FormGroup>
      <Button type="submit" isDisabled={busy || !key.trim() || !title.trim()}>Register Forgejo Git key</Button>
    </Form>
  </section>;
}
export function People({ session }: { session: Session }) {
  const [search, setSearch] = useSearchParams(); const page = search.get("page") ?? "1";
  const [items, setItems] = useState<NativeUser[] | null>(null); const [next, setNext] = useState<number | null>(null);
  const [error, setError] = useState(""); const [notice, setNotice] = useState(""); const [busy, setBusy] = useState(false); const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    const controller = new AbortController(); setItems(null); setError("");
    request<{ items: NativeUser[]; next_page: number | null }>(`/api/forgejo/admin/users?page=${encodeURIComponent(page)}`, { signal: controller.signal }).then(result => { if (!controller.signal.aborted) { setItems(result.items); setNext(result.next_page); } }).catch(failure => {
      if (controller.signal.aborted) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Forgejo People unavailable.");
    });
    return () => controller.abort();
  }, [session, page, attempt]);
  async function create(form: HTMLFormElement) {
    if (busy) return;
    const data = new FormData(form); setBusy(true); setError(""); setNotice("");
    const body = { login: String(data.get("login") ?? ""), email: String(data.get("email") ?? ""), password: String(data.get("password") ?? "") };
    // The password is local to this submission, never Zustand/browser storage.
    form.reset();
    try {
      const person = await request<NativeUser>("/api/forgejo/admin/users", { method: "POST", csrf: session.csrf_token, body });
      if (useSession.getState().session !== session) return;
      setNotice(`Forgejo created ${person.login}. They must change the initial password in native Forgejo and sign into Soda themselves.`); setAttempt(value => value + 1);
    } catch (failure) {
      if (useSession.getState().session !== session) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Creation not confirmed. Inspect native People before retrying.");
    } finally { body.password = ""; data.delete("password"); setBusy(false); }
  }
  return <section aria-busy={busy}><h1>Forgejo People</h1><p>This is the upstream user inventory. Only Forgejo administrators with explicit admin consent can use these operations.</p>
    <p><a href="/login?return_to=%2Fapp%2F&administration=1">Sign in requesting administrator consent</a>. If Forgejo reuses older consent, revoke the Soda grant in <a href={`${session.forgejo_url}/user/settings/applications`}>native Applications</a> first. This does not revoke existing Linux access.</p>
    {error && <Alert isInline variant="danger" title={error} />}{notice && <Alert isInline variant="success" title={notice} />}
    {!items && !error && <Spinner aria-label="Loading upstream People" />}
    <ul>{items?.map(person => <li key={person.id}>{person.login} — {person.full_name} {person.is_admin && "(Forgejo administrator)"}</li>)}</ul>
    {items?.length === 0 && <p>No users on this page.</p>}
    <Button variant="secondary" isDisabled={busy || Number(page) <= 1} onClick={() => setSearch({ page: String(Number(page) - 1) })}>Previous page</Button><Button variant="secondary" isDisabled={busy || next === null} onClick={() => setSearch({ page: String(next) })}>Next page</Button>
    {items && <Form onSubmit={event => { event.preventDefault(); void create(event.currentTarget); }}>
      <FormGroup label="Forgejo username" fieldId="person-login" isRequired><TextInput id="person-login" name="login" isRequired isDisabled={busy} maxLength={255} autoComplete="off" /></FormGroup>
      <FormGroup label="Email" fieldId="person-email" isRequired><TextInput id="person-email" name="email" type="email" isRequired isDisabled={busy} maxLength={320} autoComplete="off" /></FormGroup>
      <FormGroup label="Initial password — native first-login change required" fieldId="person-password" isRequired><TextInput id="person-password" name="password" type="password" isRequired isDisabled={busy} autoComplete="new-password" maxLength={4096} /></FormGroup>
      <Button type="submit" isDisabled={busy} isLoading={busy}>Create person in Forgejo</Button>
    </Form>}
  </section>;
}
