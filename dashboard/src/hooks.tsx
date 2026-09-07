import { useEffect, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { Alert, Button, Form, FormGroup, Spinner, TextInput } from "@patternfly/react-core";
import { APIError, request, type Session } from "./api";
import { useSession } from "./session";
import { repositoryPath } from "./repositories";
interface Hook { id: string; type: string; active: boolean; events: string[]; branch_filter: string; target_configured: boolean; has_authorization: boolean }
export function Hooks({ session }: { session: Session }) {
  const { owner = "", repo = "" } = useParams(); const [query, setQuery] = useSearchParams(); const page = query.get("page") ?? "1"; const api = `/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/hooks`;
  const [items, setItems] = useState<Hook[] | null>(null); const [next, setNext] = useState<number | null>(null); const [selected, setSelected] = useState<Hook | null>(null); const [attempt, setAttempt] = useState(0);
  const [url, setURL] = useState(""); const [authorization, setAuthorization] = useState(""); const [secret, setSecret] = useState(""); const [replaceAuthorization, setReplaceAuthorization] = useState(false); const [events, setEvents] = useState("push"); const [filter, setFilter] = useState(""); const [active, setActive] = useState(false);
  const [error, setError] = useState(""); const [busy, setBusy] = useState(false);
  useEffect(() => {
    const controller = new AbortController(); setItems(null); setError("");
    request<{ items: Hook[]; next_page: number | null }>(`${api}?page=${encodeURIComponent(page)}`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) { setItems(value.items); setNext(value.next_page); } }).catch(failure => { if (controller.signal.aborted) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Hook metadata unavailable."); });
    return () => controller.abort();
  }, [api, page, attempt, session]);
  function select(hook: Hook | null) { setSelected(hook); setURL(""); setAuthorization(""); setSecret(""); setReplaceAuthorization(false); setEvents(hook?.events.join(", ") ?? "push"); setFilter(hook?.branch_filter ?? ""); setActive(hook?.active ?? false); }
  async function save() {
    if (busy) return; setBusy(true); setError("");
    const fields = { events: events.split(",").map(value => value.trim()).filter(Boolean), branch_filter: filter, active };
    try {
      if (selected) await request(`${api}/${selected.id}`, { method: "PATCH", csrf: session.csrf_token, body: { ...fields, ...(url ? { url } : {}), ...(replaceAuthorization ? { authorization } : {}) } });
      else await request(api, { method: "POST", csrf: session.csrf_token, body: { ...fields, url, authorization, secret } });
      if (useSession.getState().session === session) { select(null); setAttempt(value => value + 1); }
    } catch (failure) { if (useSession.getState().session !== session) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Hook change not confirmed. Inspect native settings before retrying."); } finally { setBusy(false); }
  }
  return <section aria-busy={busy}><h1>Repository webhooks: {owner}/{repo}</h1><Link to={repositoryPath(owner, repo)}>Repository</Link>{error && <Alert isInline variant="danger" title={error} />}{!items && !error && <Spinner aria-label="Loading hook metadata" />}
    <p>Forgejo owns hook delivery and its configured host restrictions. Soda never calls hook targets. URLs, authorization headers and signing secrets are not read back; URLs can themselves contain credentials in their query.</p>
    <ul>{items?.map(hook => <li key={hook.id}>#{hook.id} {hook.type} — {hook.active ? "active" : "inactive"} — {hook.events.join(", ")} — authorization {hook.has_authorization ? "configured (hidden)" : "not configured"}<Button variant="link" isDisabled={busy} onClick={() => select(hook)}>Edit hook {hook.id}</Button></li>)}</ul>
    <Button variant="secondary" isDisabled={busy || Number(page) <= 1} onClick={() => setQuery({ page: String(Number(page) - 1) })}>Previous page</Button><Button variant="secondary" isDisabled={busy || next === null} onClick={() => next !== null && setQuery({ page: String(next) })}>Next page</Button>
    <h2>{selected ? `Edit hook ${selected.id}` : "Create Forgejo JSON webhook"}</h2>{selected && <Button variant="link" isDisabled={busy} onClick={() => select(null)}>Switch to new hook</Button>}
    <Form onSubmit={event => { event.preventDefault(); void save(); }}>
      <FormGroup label={selected ? "Replacement target URL (blank preserves hidden current URL)" : "Target HTTP(S) URL"} fieldId="hook-url"><TextInput id="hook-url" type="password" autoComplete="off" value={url} isDisabled={busy} onChange={(_, value) => setURL(value)} /></FormGroup>
      {selected && <label><input type="checkbox" checked={replaceAuthorization} disabled={busy} onChange={event => setReplaceAuthorization(event.currentTarget.checked)} /> Replace authorization header (an empty replacement clears it)</label>}
      <FormGroup label="Authorization header value (write-only)" fieldId="hook-authorization"><TextInput id="hook-authorization" type="password" autoComplete="new-password" value={authorization} isDisabled={busy || (Boolean(selected) && !replaceAuthorization)} onChange={(_, value) => setAuthorization(value)} /></FormGroup>
      {!selected && <FormGroup label="Initial signing secret (write-only)" fieldId="hook-secret"><TextInput id="hook-secret" type="password" autoComplete="new-password" value={secret} isDisabled={busy} onChange={(_, value) => setSecret(value)} /></FormGroup>}
      <FormGroup label="Native event names (comma separated)" fieldId="hook-events"><TextInput id="hook-events" value={events} isDisabled={busy} onChange={(_, value) => setEvents(value)} /></FormGroup><FormGroup label="Native branch filter" fieldId="hook-filter"><TextInput id="hook-filter" value={filter} isDisabled={busy} onChange={(_, value) => setFilter(value)} /></FormGroup>
      <label><input type="checkbox" checked={active} disabled={busy} onChange={event => setActive(event.currentTarget.checked)} /> Enable native delivery for these events</label><Button type="submit" isDisabled={busy || (!selected && !url) || !events.trim()}>{selected ? "Save native hook" : "Create native hook"}</Button>
    </Form><p>Metadata edits follow native last-write behavior. The pinned REST PATCH cannot rotate signing secrets or change package/action event flags. Those operations and other provider-specific hook creation still require integration in Soda. No hook test or delivery replay is triggered here.</p>
  </section>;
}
