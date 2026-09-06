import { useEffect, useRef, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { Alert, Button, Form, FormGroup, FormSelect, FormSelectOption, Spinner, TextArea, TextInput } from "@patternfly/react-core";
import { APIError, request, type Session } from "./api";
import { useSession } from "./session";
import { repositoryPath } from "./repositories";
import { Markdown } from "./markdown";
import { IssueActivity } from "./issue-activity";
import { Conversation } from "./conversation";

export interface Label { id: string; name: string; color: string; description: string }
export interface Milestone { id: string; title: string; description: string; state: string; due_on: string }
interface Person { id: string; login: string }
interface Issue { id: string; number: string; title: string; body: string; state: string; user: Person; assignees: Person[]; labels: Label[]; milestone: Milestone | null; updated_at: string }
export function Issues({ session }: { session: Session }) {
  const { owner = "", repo = "" } = useParams(); const route = `${repositoryPath(owner, repo)}/issues`;
  const [query, setQuery] = useSearchParams(); const queryString = query.toString();
  const [search, setSearch] = useState(query.get("q") ?? ""); const [state, setState] = useState(query.get("state") ?? "open");
  const [labels, setLabels] = useState(query.get("labels") ?? ""); const [milestones, setMilestones] = useState(query.get("milestones") ?? ""); const [assignee, setAssignee] = useState(query.get("assigned_by") ?? "");
  const [items, setItems] = useState<Issue[] | null>(null); const [next, setNext] = useState<number | null>(null); const [error, setError] = useState(""); const page = Number(query.get("page") ?? "1");
  useEffect(() => {
    const controller = new AbortController(); setItems(null); setError("");
    const current = new URLSearchParams(queryString); setSearch(current.get("q") ?? ""); setState(current.get("state") ?? "open"); setLabels(current.get("labels") ?? ""); setMilestones(current.get("milestones") ?? ""); setAssignee(current.get("assigned_by") ?? "");
    request<{ items: Issue[]; next_page: number | null }>(`/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/issues?${queryString}`, { signal: controller.signal }).then(result => { if (!controller.signal.aborted) { setItems(result.items); setNext(result.next_page); } }).catch(failure => {
      if (controller.signal.aborted) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Issues unavailable.");
    }); return () => controller.abort();
  }, [owner, repo, queryString, session]);
  function turn(page: number) { const params = new URLSearchParams(query); params.set("page", String(page)); setQuery(params); }
  return <section><h1>Issues: {owner}/{repo}</h1><nav aria-label="Issue navigation"><Link to={repositoryPath(owner, repo)}>Repository</Link><Link to={`${route}/new`}>New issue</Link><Link to={`${repositoryPath(owner, repo)}/labels`}>Labels</Link><Link to={`${repositoryPath(owner, repo)}/milestones`}>Milestones</Link></nav>
    <Form onSubmit={event => { event.preventDefault(); setQuery({ q: search, state, labels, milestones, assigned_by: assignee }); }}>
      <FormGroup label="Search issues" fieldId="issue-search"><TextInput id="issue-search" value={search} onChange={(_, value) => setSearch(value)} /></FormGroup>
      <FormGroup label="State" fieldId="issue-state"><FormSelect id="issue-state" value={state} onChange={(_, value) => setState(value)}>{["open", "closed", "all"].map(value => <FormSelectOption key={value} value={value} label={value} />)}</FormSelect></FormGroup>
      <FormGroup label="Label names (comma separated)" fieldId="issue-filter-labels"><TextInput id="issue-filter-labels" value={labels} onChange={(_, value) => setLabels(value)} /></FormGroup>
      <FormGroup label="Milestone names or IDs (comma separated)" fieldId="issue-filter-milestones"><TextInput id="issue-filter-milestones" value={milestones} onChange={(_, value) => setMilestones(value)} /></FormGroup>
      <FormGroup label="Assigned username" fieldId="issue-filter-assignee"><TextInput id="issue-filter-assignee" value={assignee} onChange={(_, value) => setAssignee(value)} /></FormGroup><Button type="submit">Apply native filters</Button>
    </Form>{error && <Alert isInline variant="danger" title={error} />}{!items && !error && <Spinner aria-label="Loading issues" />}
    <ul>{items?.map(issue => <li key={issue.id}><Link to={`${route}/${issue.number}`}>#{issue.number} {issue.title}</Link> — {issue.state} — {issue.user.login}</li>)}</ul>{items?.length === 0 && <p>No matching issues on this page.</p>}
    <Button variant="secondary" isDisabled={page <= 1} onClick={() => turn(page - 1)}>Previous page</Button><Button variant="secondary" isDisabled={next === null} onClick={() => next !== null && turn(next)}>Next page</Button>
  </section>;
}
export function NewIssue({ session }: { session: Session }) {
  const { owner = "", repo = "" } = useParams(); const navigate = useNavigate(); const active = useRef(true);
  useEffect(() => { active.current = true; return () => { active.current = false; }; }, []);
  const [title, setTitle] = useState(""); const [body, setBody] = useState(""); const [assignees, setAssignees] = useState(""); const [busy, setBusy] = useState(false); const [error, setError] = useState("");
  async function create() {
    if (busy) return; setBusy(true); setError("");
    try {
      const issue = await request<Issue>(`/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/issues`, { method: "POST", csrf: session.csrf_token, body: { title, body, assignees: assignees.split(",").map(value => value.trim()).filter(Boolean) } });
      if (active.current && useSession.getState().session === session) navigate(`${repositoryPath(owner, repo)}/issues/${issue.number}`);
    } catch (failure) {
      if (!active.current || useSession.getState().session !== session) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Issue creation not confirmed. Inspect native issues before retrying.");
    } finally { setBusy(false); }
  }
  return <section aria-busy={busy}><h1>New issue: {owner}/{repo}</h1><Link to={`${repositoryPath(owner, repo)}/issues`}>Issues</Link>{error && <Alert isInline variant="danger" title={error} />}
    <Form onSubmit={event => { event.preventDefault(); void create(); }}>
      <FormGroup label="Title" fieldId="new-issue-title" isRequired><TextInput id="new-issue-title" value={title} isRequired maxLength={255} isDisabled={busy} onChange={(_, value) => setTitle(value)} /></FormGroup>
      <FormGroup label="Markdown body" fieldId="new-issue-body"><TextArea id="new-issue-body" value={body} rows={12} isDisabled={busy} onChange={(_, value) => setBody(value)} /></FormGroup>
      <FormGroup label="Assignee usernames (comma separated, native permissions apply)" fieldId="new-issue-assignees"><TextInput id="new-issue-assignees" value={assignees} isDisabled={busy} onChange={(_, value) => setAssignees(value)} /></FormGroup>
      <Button type="submit" isDisabled={busy || !title.trim()} isLoading={busy}>Create issue in Forgejo</Button>
    </Form><h2>Preview</h2><Markdown text={body} /><p>Native structured issue-template/attachment support is not yet connected in this source view. No template validation parity is claimed.</p>
  </section>;
}
export function IssueDetail({ session }: { session: Session }) {
  const { owner = "", repo = "", index = "" } = useParams(); const route = repositoryPath(owner, repo); const api = `/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}`;
  const [issue, setIssue] = useState<Issue | null>(null);
  const [title, setTitle] = useState(""); const [body, setBody] = useState("");
  const [assignees, setAssignees] = useState(""); const [labels, setLabels] = useState(""); const [milestone, setMilestone] = useState("0");
  const dirty = useRef({ text: false, metadata: false, labels: false }); const [busy, setBusy] = useState(false); const [attempt, setAttempt] = useState(0); const [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController(); setError("");
    request<Issue>(`${api}/issues/${encodeURIComponent(index)}`, { signal: controller.signal }).then(value => {
      if (controller.signal.aborted) return; setIssue(value);
      if (!dirty.current.text) { setTitle(value.title); setBody(value.body); }
      if (!dirty.current.metadata) { setAssignees(value.assignees.map(person => person.login).join(", ")); setMilestone(value.milestone?.id ?? "0"); }
      if (!dirty.current.labels) setLabels(value.labels.map(label => label.id).join(", "));
    }).catch(failure => {
      if (controller.signal.aborted) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Issue conversation unavailable.");
    }); return () => controller.abort();
  }, [api, index, attempt, session]);
  async function mutate(action: "edit" | "state" | "labels" | "metadata") {
    if (busy || !issue) return; setBusy(true); setError("");
    let target = `${api}/issues/${index}`; let method: "POST" | "PATCH" | "PUT" = "PATCH"; let payload: unknown;
    if (action === "state") payload = { state: issue.state === "open" ? "closed" : "open" };
    else if (action === "labels") { target += "/labels"; method = "PUT"; payload = { labels: labels.split(",").map(value => value.trim()).filter(Boolean) }; }
    else if (action === "metadata") payload = { assignees: assignees.split(",").map(value => value.trim()).filter(Boolean), milestone };
    else payload = { title, body };
    try {
      await request(target, { method, csrf: session.csrf_token, body: payload });
      if (useSession.getState().session !== session) return;
      if (action === "edit") dirty.current.text = false;
      if (action === "metadata") dirty.current.metadata = false;
      if (action === "labels") dirty.current.labels = false;
      setAttempt(value => value + 1);
    } catch (failure) {
      if (useSession.getState().session !== session) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Operation not confirmed. Reload native state before retrying.");
    } finally { setBusy(false); }
  }
  return <section aria-busy={busy}><h1>Issue #{index}{issue && `: ${issue.title}`}</h1><Link to={`${route}/issues`}>Issues</Link>{error && <Alert isInline variant="danger" title={error} />}{!issue && !error && <Spinner aria-label="Loading issue" />}
    {issue && <><p>{issue.state} — opened by {issue.user.login} — updated {issue.updated_at}</p><Markdown text={issue.body} /><p>Assignees: {issue.assignees.map(person => person.login).join(", ") || "None"}. Milestone: {issue.milestone?.title ?? "None"}. Labels: {issue.labels.map(label => label.name).join(", ") || "None"}.</p>
      <IssueActivity session={session} owner={owner} repo={repo} index={index} />
      <Button isDisabled={busy} onClick={() => void mutate("state")}>{issue.state === "open" ? "Close issue" : "Reopen issue"}</Button>
      <details><summary>Edit issue and native metadata</summary><p>Forgejo enforces edit permissions. Issue edits use native last-write behavior, not a fabricated timestamp precondition.</p>
        <Form onSubmit={event => { event.preventDefault(); void mutate("edit"); }}><FormGroup label="Issue title" fieldId="edit-issue-title"><TextInput id="edit-issue-title" value={title} isDisabled={busy} onChange={(_, value) => { dirty.current.text = true; setTitle(value); }} /></FormGroup><FormGroup label="Issue body" fieldId="edit-issue-body"><TextArea id="edit-issue-body" value={body} isDisabled={busy} onChange={(_, value) => { dirty.current.text = true; setBody(value); }} /></FormGroup><Button type="submit" isDisabled={busy}>Save issue text</Button></Form>
        <Form onSubmit={event => { event.preventDefault(); void mutate("metadata"); }}><FormGroup label="Assignee usernames (comma separated)" fieldId="edit-issue-assignees"><TextInput id="edit-issue-assignees" value={assignees} isDisabled={busy} onChange={(_, value) => { dirty.current.metadata = true; setAssignees(value); }} /></FormGroup><FormGroup label="Milestone ID (0 removes it)" fieldId="edit-issue-milestone"><TextInput id="edit-issue-milestone" value={milestone} isDisabled={busy} onChange={(_, value) => { dirty.current.metadata = true; setMilestone(value); }} /></FormGroup><Link to={`${route}/milestones`}>Browse native milestone IDs</Link><Button type="submit" isDisabled={busy}>Save assignments and milestone</Button></Form>
        <Form onSubmit={event => { event.preventDefault(); void mutate("labels"); }}><FormGroup label="Label IDs (comma separated, empty removes labels)" fieldId="edit-issue-labels"><TextInput id="edit-issue-labels" value={labels} isDisabled={busy} onChange={(_, value) => { dirty.current.labels = true; setLabels(value); }} /></FormGroup><Link to={`${route}/labels`}>Browse native label IDs</Link><Button type="submit" isDisabled={busy}>Replace issue labels</Button></Form>
      </details><Conversation session={session} owner={owner} repo={repo} index={index} />
    </>}
  </section>;
}
