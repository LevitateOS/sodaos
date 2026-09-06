import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { Alert, Button, Form, FormGroup, Spinner, TextArea } from "@patternfly/react-core";
import { APIError, request, type Session } from "./api";
import { useSession } from "./session";
import { Markdown } from "./markdown";
interface Comment { id: string; body: string; user: { id: string; login: string }; created_at: string }
// Forgejo issues and pull requests share the same native conversation API.
export function Conversation({ session, owner, repo, index }: { session: Session; owner: string; repo: string; index: string }) {
  const [query, setQuery] = useSearchParams(); const page = query.get("comments_page") ?? "1";
  const [items, setItems] = useState<Comment[] | null>(null); const [next, setNext] = useState<number | null>(null); const [error, setError] = useState("");
  const [body, setBody] = useState(""); const [editing, setEditing] = useState<string | null>(null); const [busy, setBusy] = useState(false); const [attempt, setAttempt] = useState(0);
  const api = `/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}`;
  useEffect(() => {
    const controller = new AbortController(); setItems(null); setError("");
    request<{ items: Comment[]; next_page: number | null }>(`${api}/issues/${encodeURIComponent(index)}/comments?page=${encodeURIComponent(page)}`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) { setItems(value.items); setNext(value.next_page); } }).catch(failure => {
      if (controller.signal.aborted) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Conversation unavailable.");
    }); return () => controller.abort();
  }, [api, index, page, attempt, session]);
  async function save() {
    if (busy || !body.trim()) return; setBusy(true); setError("");
    try {
      await request(editing ? `${api}/issue-comments/${editing}` : `${api}/issues/${index}/comments`, { method: editing ? "PATCH" : "POST", body: { body }, csrf: session.csrf_token });
      if (useSession.getState().session === session) { setEditing(null); setBody(""); setAttempt(value => value + 1); }
    } catch (failure) {
      if (useSession.getState().session !== session) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Comment not confirmed; inspect conversation before retrying.");
    } finally { setBusy(false); }
  }
  function turn(page: number) { const nextQuery = new URLSearchParams(query); nextQuery.set("comments_page", String(page)); setQuery(nextQuery); }
  return <section aria-busy={busy} aria-label="Conversation"><h2>Conversation</h2>{error && <Alert isInline variant="danger" title={error} />}{!items && !error && <Spinner aria-label="Loading comments" />}
    {items?.map(item => <article key={item.id}><h3><Link to={`/users/${encodeURIComponent(item.user.login)}`}>{item.user.login}</Link> — {item.created_at}</h3><Markdown text={item.body} />{item.user.id === session.user.id && <Button variant="link" isDisabled={busy} onClick={() => { setEditing(item.id); setBody(item.body); }}>Edit this comment</Button>}</article>)}
    <Button variant="secondary" isDisabled={busy || Number(page) <= 1} onClick={() => turn(Number(page) - 1)}>Previous comments</Button><Button variant="secondary" isDisabled={busy || next === null} onClick={() => next !== null && turn(next)}>Next comments</Button>
    <Form onSubmit={event => { event.preventDefault(); void save(); }}><FormGroup label={editing ? "Edit comment" : "New comment"} fieldId="conversation-comment"><TextArea id="conversation-comment" value={body} isDisabled={busy} rows={8} onChange={(_, value) => setBody(value)} /></FormGroup><Button type="submit" isDisabled={busy || !body.trim()}>{editing ? "Save comment" : "Post comment"}</Button>{editing && <Button variant="link" isDisabled={busy} onClick={() => { setEditing(null); setBody(""); }}>Cancel edit</Button>}</Form>
  </section>;
}
