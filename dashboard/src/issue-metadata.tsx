import { useEffect, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { Alert, Button, Form, FormGroup, FormSelect, FormSelectOption, Spinner, TextArea, TextInput } from "@patternfly/react-core";
import { APIError, request, type Session } from "./api";
import { useSession } from "./session";
import { repositoryPath } from "./repositories";
import type { Label, Milestone } from "./issues";

export function Labels({ session }: { session: Session }) {
  const { owner = "", repo = "" } = useParams(); const route = repositoryPath(owner, repo); const api = `/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/labels`;
  const [query, setQuery] = useSearchParams(); const page = query.get("page") ?? "1";
  const [items, setItems] = useState<Label[] | null>(null); const [next, setNext] = useState<number | null>(null); const [error, setError] = useState(""); const [attempt, setAttempt] = useState(0); const [busy, setBusy] = useState(false);
  const [editing, setEditing] = useState<string | null>(null); const [name, setName] = useState(""); const [color, setColor] = useState("0088cc"); const [description, setDescription] = useState("");
  useEffect(() => {
    const controller = new AbortController(); setItems(null); setError("");
    request<{ items: Label[]; next_page: number | null }>(`${api}?page=${encodeURIComponent(page)}`, { signal: controller.signal }).then(result => { if (!controller.signal.aborted) { setItems(result.items); setNext(result.next_page); } }).catch(failure => {
      if (controller.signal.aborted) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Labels unavailable.");
    }); return () => controller.abort();
  }, [api, page, attempt, session]);
  async function save() {
    if (busy) return; setBusy(true); setError("");
    try {
      await request(editing ? `${api}/${editing}` : api, { method: editing ? "PATCH" : "POST", csrf: session.csrf_token, body: { name, color, description } });
      if (useSession.getState().session === session) { setEditing(null); setName(""); setDescription(""); setAttempt(value => value + 1); }
    } catch (failure) {
      if (useSession.getState().session !== session) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Label change not confirmed. Refresh native state before retrying.");
    } finally { setBusy(false); }
  }
  return <section aria-busy={busy}><h1>Labels: {owner}/{repo}</h1><Link to={`${route}/issues`}>Issues</Link>{error && <Alert isInline variant="danger" title={error} />}{!items && !error && <Spinner aria-label="Loading labels" />}
    <ul>{items?.map(label => <li key={label.id}><Link to={`${route}/issues?${new URLSearchParams({ labels: label.name })}`}>{label.name}</Link> — ID <code>{label.id}</code> — #{label.color}<p>{label.description}</p><Button variant="link" isDisabled={busy} onClick={() => { setEditing(label.id); setName(label.name); setColor(label.color.replace(/^#/, "")); setDescription(label.description); }}>Edit {label.name}</Button></li>)}</ul>
    <Button variant="secondary" isDisabled={busy || Number(page) <= 1} onClick={() => setQuery({ page: String(Number(page) - 1) })}>Previous page</Button><Button variant="secondary" isDisabled={busy || next === null} onClick={() => setQuery({ page: String(next) })}>Next page</Button>
    <h2>{editing ? `Edit label ${editing}` : "Create label"}</h2><Form onSubmit={event => { event.preventDefault(); void save(); }}>
      <FormGroup label="Label name" fieldId="label-name" isRequired><TextInput id="label-name" value={name} isRequired isDisabled={busy} maxLength={255} onChange={(_, value) => setName(value)} /></FormGroup>
      <FormGroup label="Color (six hex digits, without #)" fieldId="label-color" isRequired><TextInput id="label-color" value={color} isRequired isDisabled={busy} maxLength={6} onChange={(_, value) => setColor(value)} /></FormGroup>
      <FormGroup label="Description" fieldId="label-description"><TextArea id="label-description" value={description} isDisabled={busy} onChange={(_, value) => setDescription(value)} /></FormGroup>
      <Button type="submit" isDisabled={busy || !name.trim()}>{editing ? "Save label" : "Create label"}</Button>{editing && <Button variant="link" isDisabled={busy} onClick={() => { setEditing(null); setName(""); setDescription(""); }}>Cancel edit</Button>}
    </Form><p>Forgejo requires native issue/pull write permission. Label edits do not affect environments.</p>
  </section>;
}
export function Milestones({ session }: { session: Session }) {
  const { owner = "", repo = "" } = useParams(); const route = repositoryPath(owner, repo); const api = `/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/milestones`;
  const [query, setQuery] = useSearchParams(); const page = query.get("page") ?? "1";
  const [items, setItems] = useState<Milestone[] | null>(null); const [next, setNext] = useState<number | null>(null); const [error, setError] = useState(""); const [attempt, setAttempt] = useState(0); const [busy, setBusy] = useState(false);
  const [editing, setEditing] = useState<string | null>(null); const [title, setTitle] = useState(""); const [description, setDescription] = useState(""); const [state, setState] = useState("open"); const [due, setDue] = useState("");
  useEffect(() => {
    const controller = new AbortController(); setItems(null); setError("");
    request<{ items: Milestone[]; next_page: number | null }>(`${api}?page=${encodeURIComponent(page)}`, { signal: controller.signal }).then(result => { if (!controller.signal.aborted) { setItems(result.items); setNext(result.next_page); } }).catch(failure => {
      if (controller.signal.aborted) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Milestones unavailable.");
    }); return () => controller.abort();
  }, [api, page, attempt, session]);
  async function save() {
    if (busy) return; setBusy(true); setError("");
    try {
      await request(editing ? `${api}/${editing}` : api, { method: editing ? "PATCH" : "POST", csrf: session.csrf_token, body: { title, description, state, due_on: due } });
      if (useSession.getState().session === session) { setEditing(null); setTitle(""); setDescription(""); setDue(""); setAttempt(value => value + 1); }
    } catch (failure) {
      if (useSession.getState().session !== session) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Milestone change not confirmed. Refresh native state before retrying.");
    } finally { setBusy(false); }
  }
  return <section aria-busy={busy}><h1>Milestones: {owner}/{repo}</h1><Link to={`${route}/issues`}>Issues</Link>{error && <Alert isInline variant="danger" title={error} />}{!items && !error && <Spinner aria-label="Loading milestones" />}
    <ul>{items?.map(item => <li key={item.id}><Link to={`${route}/issues?${new URLSearchParams({ milestones: item.id, state: "all" })}`}>{item.title}</Link> — ID <code>{item.id}</code> — {item.state}<p>{item.description}</p><p>Due: {item.due_on || "None"}</p><Button variant="link" isDisabled={busy} onClick={() => { setEditing(item.id); setTitle(item.title); setDescription(item.description); setState(item.state); setDue(item.due_on); }}>Edit {item.title}</Button></li>)}</ul>
    <Button variant="secondary" isDisabled={busy || Number(page) <= 1} onClick={() => setQuery({ page: String(Number(page) - 1) })}>Previous page</Button><Button variant="secondary" isDisabled={busy || next === null} onClick={() => setQuery({ page: String(next) })}>Next page</Button>
    <h2>{editing ? `Edit milestone ${editing}` : "Create milestone"}</h2><Form onSubmit={event => { event.preventDefault(); void save(); }}>
      <FormGroup label="Milestone title" fieldId="milestone-title" isRequired><TextInput id="milestone-title" value={title} isRequired isDisabled={busy} maxLength={255} onChange={(_, value) => setTitle(value)} /></FormGroup>
      <FormGroup label="Description" fieldId="milestone-description"><TextArea id="milestone-description" value={description} isDisabled={busy} onChange={(_, value) => setDescription(value)} /></FormGroup>
      <FormGroup label="State" fieldId="milestone-state"><FormSelect id="milestone-state" value={state} isDisabled={busy} onChange={(_, value) => setState(value)}><FormSelectOption value="open" label="Open" /><FormSelectOption value="closed" label="Closed" /></FormSelect></FormGroup>
      <FormGroup label="Due date (RFC3339; empty leaves an existing date unchanged)" fieldId="milestone-due"><TextInput id="milestone-due" value={due} isDisabled={busy} placeholder="2026-12-31T12:00:00Z" onChange={(_, value) => setDue(value)} /></FormGroup>
      <Button type="submit" isDisabled={busy || !title.trim()}>{editing ? "Save milestone" : "Create milestone"}</Button>{editing && <Button variant="link" isDisabled={busy} onClick={() => { setEditing(null); setTitle(""); setDescription(""); setDue(""); }}>Cancel edit</Button>}
    </Form>
  </section>;
}
