import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { Alert, Button, Spinner } from "@patternfly/react-core";
import { APIError, request, type Session } from "./api";
import { useSession } from "./session";
interface Notification { id: string; title: string; unread: boolean; pinned: boolean; updated_at: string; route: string; repository: { full_name: string } }
export function Notifications({ session }: { session: Session }) {
  const [query, setQuery] = useSearchParams(); const page = query.get("page") ?? "1"; const all = query.get("all") ?? "false";
  const [items, setItems] = useState<Notification[] | null>(null); const [next, setNext] = useState<number | null>(null); const [attempt, setAttempt] = useState(0); const [busy, setBusy] = useState(false); const [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController(); setItems(null); setError(""); setNext(null);
    request<{ items: Notification[]; next_page: number | null }>(`/api/forgejo/notifications?${new URLSearchParams({ all, page })}`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) { setItems(value.items); setNext(value.next_page); } }).catch(failure => { if (controller.signal.aborted) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Native inbox unavailable."); }); return () => controller.abort();
  }, [all, page, attempt, session]);
  async function mark(id: string, state: string) {
    if (busy) return; setBusy(true); setError("");
    try { await request(`/api/forgejo/notifications/${id}`, { method: "PATCH", body: { state }, csrf: session.csrf_token }); if (useSession.getState().session === session) setAttempt(value => value + 1); }
    catch (failure) { if (useSession.getState().session !== session) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Native notification update not confirmed."); } finally { setBusy(false); }
  }
  return <section aria-busy={busy}><h1>Native notifications</h1><label><input type="checkbox" checked={all === "true"} disabled={busy} onChange={event => setQuery({ all: String(event.currentTarget.checked) })} /> Include read notifications</label>{error && <Alert isInline variant="danger" title={error} />}{!items && !error && <Spinner aria-label="Loading native inbox" />}
    <ul>{items?.map(item => <li key={item.id}>{item.route ? <Link to={item.route}>{item.title}</Link> : item.title} — {item.repository.full_name} — {item.unread ? "unread" : "read"}{item.pinned && " (pinned)"} — {item.updated_at}<Button variant="link" isDisabled={busy} onClick={() => void mark(item.id, item.unread ? "read" : "unread")}>{item.unread ? "Mark read" : "Mark unread"}</Button>{!item.pinned && <Button variant="link" isDisabled={busy} onClick={() => void mark(item.id, "pinned")}>Pin notification</Button>}</li>)}</ul>{items?.length === 0 && <p>No notifications on this native page.</p>}
    <Button variant="secondary" isDisabled={busy || Number(page) <= 1} onClick={() => setQuery({ all, page: String(Number(page) - 1) })}>Previous page</Button><Button variant="secondary" isDisabled={busy || next === null} onClick={() => next !== null && setQuery({ all, page: String(next) })}>Next page</Button>
  </section>;
}
