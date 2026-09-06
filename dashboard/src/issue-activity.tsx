import { useEffect, useRef, useState } from "react";
import { Alert, Button, Form, FormGroup, TextInput } from "@patternfly/react-core";
import { APIError, request, type Session } from "./api";
import { useSession } from "./session";

interface Reaction { content: string; user: { id: string; login: string } }
interface Attachment { id: string; name: string; size: string; native_url: string }
export function IssueActivity({ session, owner, repo, index }: { session: Session; owner: string; repo: string; index: string }) {
  const active = useRef(true);
  useEffect(() => { active.current = true; return () => { active.current = false; }; }, []);
  const api = `/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/issues/${encodeURIComponent(index)}`;
  const [reactions, setReactions] = useState<Reaction[] | null>(null); const [subscribed, setSubscribed] = useState<boolean | null>(null); const [attachments, setAttachments] = useState<Attachment[] | null>(null);
  const [warnings, setWarnings] = useState<string[]>([]); const [error, setError] = useState(""); const [busy, setBusy] = useState(false); const [attempt, setAttempt] = useState(0); const [reaction, setReaction] = useState("+1");
  useEffect(() => {
    const controller = new AbortController(); setReactions(null); setSubscribed(null); setAttachments(null); setWarnings([]);
    function failed(feature: string, failure: unknown) {
      if (controller.signal.aborted) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setWarnings(current => [...current, `${feature}: ${failure instanceof Error ? failure.message : "Unavailable"}`]);
    }
    request<{ items: Reaction[] }>(`${api}/reactions`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) setReactions(value.items); }).catch(failure => failed("Reactions", failure));
    request<{ subscribed: boolean }>(`${api}/subscription`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) setSubscribed(value.subscribed); }).catch(failure => failed("Subscription", failure));
    request<{ items: Attachment[] }>(`${api}/attachments`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) setAttachments(value.items); }).catch(failure => failed("Attachments", failure));
    return () => controller.abort();
  }, [api, session, attempt]);
  async function subscription() {
    if (busy || subscribed === null) return; setBusy(true); setError("");
    try {
      await request(`${api}/subscription`, { method: subscribed ? "DELETE" : "PUT", body: {}, csrf: session.csrf_token });
      if (useSession.getState().session === session) setAttempt(value => value + 1);
    } catch (failure) { if (useSession.getState().session === session) setError(failure instanceof Error ? failure.message : "Subscription not confirmed."); } finally { setBusy(false); }
  }
  async function react(content: string, remove: boolean) {
    if (busy) return; setBusy(true); setError("");
    try {
      await request(`${api}/reactions`, { method: remove ? "DELETE" : "POST", body: { content }, csrf: session.csrf_token });
      if (useSession.getState().session === session) setAttempt(value => value + 1);
    } catch (failure) { if (useSession.getState().session === session) setError(failure instanceof Error ? failure.message : "Reaction not confirmed."); } finally { setBusy(false); }
  }
  async function upload(file?: File) {
    if (busy || !file) return;
    if (file.size > 32768) { setError("Attachment uploads are limited to 32 KiB in this view. Use native Forgejo for larger attachments."); return; }
    setBusy(true); setError("");
    try {
      const bytes = new Uint8Array(await file.arrayBuffer());
      if (!active.current || useSession.getState().session !== session) return;
      const content = btoa(Array.from(bytes, byte => String.fromCharCode(byte)).join(""));
      await request(`${api}/attachments`, { method: "POST", body: { name: file.name, content }, csrf: session.csrf_token });
      if (useSession.getState().session === session) setAttempt(value => value + 1);
    } catch (failure) { if (useSession.getState().session === session) setError(failure instanceof Error ? failure.message : "Attachment not confirmed. Inspect native attachments before retrying."); } finally { setBusy(false); }
  }
  return <section aria-busy={busy} aria-label="Issue reactions subscription and attachments">
    {warnings.map(warning => <Alert key={warning} isInline variant="warning" title={warning} />)}{error && <Alert isInline variant="danger" title={error} />}
    <h2>Notifications for this issue</h2>{subscribed !== null && <><p>{subscribed ? "Subscribed" : "Not subscribed"}</p><Button isDisabled={busy} onClick={() => void subscription()}>{subscribed ? "Unsubscribe myself" : "Subscribe myself"}</Button></>}
    <h2>Reactions</h2><ul>{reactions?.map(item => <li key={`${item.user.id}:${item.content}`}>{item.user.login}: {item.content}{item.user.id === session.user.id && <Button variant="link" isDisabled={busy} onClick={() => void react(item.content, true)}>Remove my {item.content}</Button>}</li>)}</ul>
    <Form onSubmit={event => { event.preventDefault(); void react(reaction, false); }}><FormGroup label="Native reaction name" fieldId="issue-reaction"><TextInput id="issue-reaction" value={reaction} isDisabled={busy} maxLength={64} onChange={(_, value) => setReaction(value)} /></FormGroup><Button type="submit" isDisabled={busy || !reaction}>Add my reaction</Button></Form>
    <h2>Attachments</h2><ul>{attachments?.map(item => <li key={item.id}>{item.native_url ? <a href={item.native_url}>{item.name}</a> : item.name} — {item.size} bytes</li>)}</ul>
    <p>Downloads currently leave Soda for native Forgejo authorization; this is a labeled native dependency, not a Soda binary proxy. Attachments are never rendered as active content here.</p>
    <FormGroup label="Upload issue attachment (up to 32 KiB)" fieldId="issue-attachment"><input id="issue-attachment" type="file" disabled={busy} onChange={event => { const file = event.currentTarget.files?.[0]; event.currentTarget.value = ""; void upload(file); }} /></FormGroup>
  </section>;
}
