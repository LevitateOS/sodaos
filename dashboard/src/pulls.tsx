import { useEffect, useRef, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { Alert, Button, Form, FormGroup, FormSelect, FormSelectOption, Spinner, TextArea, TextInput } from "@patternfly/react-core";
import { APIError, request, type Session } from "./api";
import { useSession } from "./session";
import { repositoryPath } from "./repositories";
import { Markdown } from "./markdown";
import { Conversation } from "./conversation";
import { PullReview } from "./pull-review";
import { PullInspection } from "./pull-inspection";
export interface Pull { id: string; number: string; title: string; body: string; state: string; user: { id: string; login: string }; head: { ref: string; sha: string; label: string }; base: { ref: string; sha: string; label: string }; merge_base: string; merged: boolean; mergeable: boolean; draft: boolean; merge_methods: string[]; requested_reviewers: { id: string; login: string }[] }
export function Pulls({ session }: { session: Session }) {
  const { owner = "", repo = "" } = useParams(); const route = `${repositoryPath(owner, repo)}/pulls`; const [query, setQuery] = useSearchParams(); const state = query.get("state") ?? "open"; const page = query.get("page") ?? "1";
  const [items, setItems] = useState<Pull[] | null>(null); const [next, setNext] = useState<number | null>(null); const [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController(); setItems(null); setError("");
    request<{ items: Pull[]; next_page: number | null }>(`/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/pulls?${new URLSearchParams({ state, page })}`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) { setItems(value.items); setNext(value.next_page); } }).catch(failure => {
      if (controller.signal.aborted) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Pull requests unavailable.");
    }); return () => controller.abort();
  }, [owner, repo, state, page, session]);
  return <section><h1>Pull requests: {owner}/{repo}</h1><nav><Link to={repositoryPath(owner, repo)}>Repository</Link><Link to={`${route}/new`}>New pull request</Link></nav>
    <FormGroup label="Pull request state" fieldId="pull-state"><FormSelect id="pull-state" value={state} onChange={(_, value) => setQuery({ state: value })}>{["open", "closed", "all"].map(value => <FormSelectOption key={value} value={value} label={value} />)}</FormSelect></FormGroup>
    {error && <Alert isInline variant="danger" title={error} />}{!items && !error && <Spinner aria-label="Loading pull requests" />}<ul>{items?.map(item => <li key={item.id}><Link to={`${route}/${item.number}`}>#{item.number} {item.title}</Link> — {item.merged ? "merged" : item.state} — {item.user.login}</li>)}</ul>
    <Button variant="secondary" isDisabled={Number(page) <= 1} onClick={() => setQuery({ state, page: String(Number(page) - 1) })}>Previous page</Button><Button variant="secondary" isDisabled={next === null} onClick={() => next !== null && setQuery({ state, page: String(next) })}>Next page</Button>
  </section>;
}
export function NewPull({ session }: { session: Session }) {
  const { owner = "", repo = "" } = useParams(); const navigate = useNavigate(); const [query] = useSearchParams(); const active = useRef(true);
  useEffect(() => { active.current = true; return () => { active.current = false; }; }, []);
  const [base, setBase] = useState(query.get("base") ?? ""); const [head, setHead] = useState(query.get("head") ?? ""); const [title, setTitle] = useState(""); const [body, setBody] = useState(""); const [busy, setBusy] = useState(false); const [error, setError] = useState("");
  async function create() {
    if (busy) return; setBusy(true); setError("");
    try {
      const value = await request<Pull>(`/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/pulls`, { method: "POST", csrf: session.csrf_token, body: { base, head, title, body } });
      if (active.current && useSession.getState().session === session) navigate(`${repositoryPath(owner, repo)}/pulls/${value.number}`);
    } catch (failure) {
      if (!active.current || useSession.getState().session !== session) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Creation not confirmed; inspect native pull requests before retrying.");
    } finally { setBusy(false); }
  }
  return <section aria-busy={busy}><h1>New pull request: {owner}/{repo}</h1><Link to={`${repositoryPath(owner, repo)}/pulls`}>Pull requests</Link>{error && <Alert isInline variant="danger" title={error} />}<Form onSubmit={event => { event.preventDefault(); void create(); }}>
    <FormGroup label="Base branch" fieldId="pull-base"><TextInput id="pull-base" value={base} isDisabled={busy} onChange={(_, value) => setBase(value)} /></FormGroup><FormGroup label="Head branch (native owner:branch for a fork)" fieldId="pull-head"><TextInput id="pull-head" value={head} isDisabled={busy} onChange={(_, value) => setHead(value)} /></FormGroup>
    <FormGroup label="Pull request title" fieldId="pull-title"><TextInput id="pull-title" value={title} maxLength={255} isDisabled={busy} onChange={(_, value) => setTitle(value)} /></FormGroup><FormGroup label="Pull request body" fieldId="pull-body"><TextArea id="pull-body" value={body} rows={12} isDisabled={busy} onChange={(_, value) => setBody(value)} /></FormGroup>
    <Button type="submit" isDisabled={busy || !base || !head || !title.trim()}>Create pull request in Forgejo</Button></Form><h2>Preview</h2><Markdown text={body} />
  </section>;
}
export function PullDetail({ session }: { session: Session }) {
  const { owner = "", repo = "", index = "" } = useParams(); const route = repositoryPath(owner, repo); const [query, setQuery] = useSearchParams(); const tab = query.get("tab") ?? "conversation";
  const [pull, setPull] = useState<Pull | null>(null); const [error, setError] = useState(""); const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    const controller = new AbortController(); setPull(null); setError("");
    request<Pull>(`/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/pulls/${encodeURIComponent(index)}`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) setPull(value); }).catch(failure => { if (controller.signal.aborted) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Pull request unavailable."); });
    return () => controller.abort();
  }, [owner, repo, index, attempt, session]);
  return <section><h1>Pull request #{index}{pull && `: ${pull.title}`}</h1><nav><Link to={`${route}/pulls`}>Pull requests</Link><Button variant="link" onClick={() => setAttempt(value => value + 1)}>Reload native pull request</Button></nav>
    {error && <Alert isInline variant="danger" title={error} />}{!pull && !error && <Spinner aria-label="Loading pull request" />}{pull && <><p>{pull.merged ? "Merged" : pull.state}{pull.draft && " (draft)"} — {pull.head.label || pull.head.ref} → {pull.base.ref}</p><p>Displayed head: <code>{pull.head.sha}</code>; base: <code>{pull.base.sha}</code>.</p><Markdown text={pull.body} />
      <nav aria-label="Pull request tabs">{["conversation", "commits", "files", "checks", "reviews"].map(value => <Button key={value} variant={value === tab ? "primary" : "secondary"} onClick={() => setQuery({ tab: value })}>{value}</Button>)}</nav>
      {tab === "conversation" && <Conversation session={session} owner={owner} repo={repo} index={index} />}
      {(tab === "commits" || tab === "files" || tab === "checks") && <PullInspection key={`${tab}:${pull.head.sha}:${pull.base.sha}`} session={session} owner={owner} repo={repo} pull={pull} tab={tab} />}
      {tab === "reviews" && <PullReview key={`${pull.head.sha}:${pull.base.sha}`} session={session} owner={owner} repo={repo} pull={pull} onChanged={() => setAttempt(value => value + 1)} />}
      <p><a href={`${session.forgejo_url}/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/pulls/${encodeURIComponent(index)}`}>Native Forgejo pull request</a> remains available for unsupported review details; no borrowed browser credentials.</p>
    </>}
  </section>;
}
