import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { Alert, Button, Form, FormGroup, FormSelect, FormSelectOption, Spinner, TextArea, TextInput } from "@patternfly/react-core";
import { APIError, request, type Session } from "./api";
import { useSession } from "./session";
import { Markdown } from "./markdown";
import type { Pull } from "./pulls";
interface Review { id: string; user: { id: string; login: string }; body: string; commit_id: string; state: string; stale: boolean; dismissed: boolean; submitted_at: string }
export function PullReview({ session, owner, repo, pull, onChanged }: { session: Session; owner: string; repo: string; pull: Pull; onChanged: () => void }) {
  const [query, setQuery] = useSearchParams(); const page = query.get("page") ?? "1"; const api = `/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/pulls/${pull.number}`;
  const [reviews, setReviews] = useState<Review[] | null>(null); const [next, setNext] = useState<number | null>(null); const [error, setError] = useState(""); const [busy, setBusy] = useState(false);
  const [body, setBody] = useState(""); const [event, setEvent] = useState("COMMENT"); const [path, setPath] = useState(""); const [line, setLine] = useState(""); const [inlineBody, setInlineBody] = useState(""); const [reviewers, setReviewers] = useState("");
  const [strategy, setStrategy] = useState(pull.merge_methods[0] ?? ""); const [mergeTitle, setMergeTitle] = useState(""); const [mergeMessage, setMergeMessage] = useState(""); const [confirmed, setConfirmed] = useState(false);
  useEffect(() => {
    const controller = new AbortController(); setReviews(null); setError("");
    request<{ items: Review[]; next_page: number | null }>(`${api}/reviews?page=${encodeURIComponent(page)}`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) { setReviews(value.items); setNext(value.next_page); } }).catch(failure => {
      if (controller.signal.aborted) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Native reviews unavailable.");
    }); return () => controller.abort();
  }, [api, page, session]);
  async function mutate(action: "review" | "request" | "cancel" | "merge") {
    if (busy) return;
    if (action === "merge" && !confirmed) return;
    setBusy(true); setError("");
    const snapshot = { head: pull.head.sha, base: pull.base.sha, merge_base: pull.merge_base };
    try {
      if (action === "review") {
        const comments = inlineBody.trim() ? [{ path, line: Number(line), body: inlineBody }] : [];
        await request(`${api}/reviews`, { method: "POST", csrf: session.csrf_token, body: { ...snapshot, event, body, comments } });
      } else if (action === "merge") {
        await request(`${api}/merge`, { method: "POST", csrf: session.csrf_token, body: { ...snapshot, strategy, title: mergeTitle, message: mergeMessage } });
      } else {
        await request(`${api}/reviewers`, { method: action === "cancel" ? "DELETE" : "POST", csrf: session.csrf_token, body: { reviewers: reviewers.split(",").map(value => value.trim()).filter(Boolean) } });
      }
      if (useSession.getState().session === session) onChanged();
    } catch (failure) {
      if (useSession.getState().session !== session) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Native operation not confirmed. Inspect the pull request and pending reviews before retrying.");
    } finally { setBusy(false); }
  }
  function turn(page: number) { const params = new URLSearchParams(query); params.set("page", String(page)); setQuery(params); }
  return <section aria-busy={busy}><h2>Native reviews</h2>{error && <Alert isInline variant="danger" title={error} />}{!reviews && !error && <Spinner aria-label="Loading reviews" />}
    {reviews?.map(review => <article key={review.id}><h3>{review.user.login}: {review.state}{review.stale && " (stale)"}{review.dismissed && " (dismissed)"}</h3><p>Commit <code>{review.commit_id}</code> — {review.submitted_at}</p><Markdown text={review.body} /></article>)}
    <Button variant="secondary" isDisabled={busy || Number(page) <= 1} onClick={() => turn(Number(page) - 1)}>Previous reviews</Button><Button variant="secondary" isDisabled={busy || next === null} onClick={() => next !== null && turn(next)}>Next reviews</Button>
    <h2>Submit review of displayed head</h2><p>Reviews always carry <code>{pull.head.sha}</code>. New-side inline comments use native file line numbers, not guessed unified-diff offsets. Old-side comment mapping, existing inline-thread rendering and team review requests remain unfinished custom source; use native Forgejo for those details. A failed submission can leave native pending comments: inspect before retrying.</p>
    <Form onSubmit={event => { event.preventDefault(); void mutate("review"); }}>
      <FormGroup label="Review decision" fieldId="review-event"><FormSelect id="review-event" value={event} isDisabled={busy} onChange={(_, value) => setEvent(value)}>{[["COMMENT", "Comment"], ["APPROVED", "Approve"], ["REQUEST_CHANGES", "Request changes"]].map(([value, label]) => <FormSelectOption key={value} value={value} label={label} />)}</FormSelect></FormGroup>
      <FormGroup label="Review body" fieldId="review-body"><TextArea id="review-body" value={body} isDisabled={busy} rows={8} onChange={(_, value) => setBody(value)} /></FormGroup>
      <details><summary>Optional new-side inline comment</summary><FormGroup label="Changed file path" fieldId="review-path"><TextInput id="review-path" value={path} isDisabled={busy} onChange={(_, value) => setPath(value)} /></FormGroup><FormGroup label="Line number in the displayed head file" fieldId="review-line"><TextInput id="review-line" type="number" min={1} max={2147483647} value={line} isDisabled={busy} onChange={(_, value) => setLine(value)} /></FormGroup><FormGroup label="Inline comment body" fieldId="review-inline"><TextArea id="review-inline" value={inlineBody} isDisabled={busy} onChange={(_, value) => setInlineBody(value)} /></FormGroup></details>
      <Button type="submit" isDisabled={busy || (Boolean(inlineBody.trim()) && (!path || !Number.isInteger(Number(line)) || Number(line) < 1))}>Submit native review</Button></Form>
    <h2>Requested reviewers</h2><p>{pull.requested_reviewers.map(person => person.login).join(", ") || "None"}</p><Form onSubmit={event => { event.preventDefault(); void mutate("request"); }}><FormGroup label="Reviewer usernames (comma separated)" fieldId="request-reviewers"><TextInput id="request-reviewers" value={reviewers} isDisabled={busy} onChange={(_, value) => setReviewers(value)} /></FormGroup><Button type="submit" isDisabled={busy || !reviewers.trim()}>Request native reviews</Button><Button variant="secondary" isDisabled={busy || !reviewers.trim()} onClick={() => void mutate("cancel")}>Cancel these review requests</Button></Form>
    <h2>Merge using native rules</h2><p>Forgejo enforces protection, checks and reviewer requirements. No force, auto-merge, branch deletion, project cleanup, tool switch or workload promotion is requested.</p>
    <Form onSubmit={event => { event.preventDefault(); void mutate("merge"); }}><FormGroup label="Enabled native merge strategy" fieldId="merge-strategy"><FormSelect id="merge-strategy" value={strategy} isDisabled={busy} onChange={(_, value) => setStrategy(value)}>{pull.merge_methods.map(value => <FormSelectOption key={value} value={value} label={value} />)}</FormSelect></FormGroup><FormGroup label="Optional merge title" fieldId="merge-title"><TextInput id="merge-title" value={mergeTitle} isDisabled={busy} maxLength={255} onChange={(_, value) => setMergeTitle(value)} /></FormGroup><FormGroup label="Optional merge message" fieldId="merge-message"><TextArea id="merge-message" value={mergeMessage} isDisabled={busy} onChange={(_, value) => setMergeMessage(value)} /></FormGroup><label><input type="checkbox" checked={confirmed} disabled={busy} onChange={event => setConfirmed(event.currentTarget.checked)} /> Merge the displayed head {pull.head.sha} into {pull.base.ref}</label><Button type="submit" isDisabled={busy || !confirmed || !strategy || pull.merged || pull.state !== "open" || pull.draft || !pull.mergeable}>Merge in Forgejo</Button></Form>
  </section>;
}
