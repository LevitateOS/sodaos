import { useEffect, useRef, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { Alert, Button, Form, FormGroup, Spinner, TextArea, TextInput } from "@patternfly/react-core";
import { APIError, request, type Session } from "./api";
import { useSession } from "./session";
import { repositoryPath } from "./repositories";

interface Commit { sha: string; created: string; commit: { message: string; author: { name: string; date: string } }; files: { filename: string; status: string }[] | null }
export function History({ session }: { session: Session }) {
  const { owner = "", repo = "" } = useParams(); const route = repositoryPath(owner, repo);
  const [query, setQuery] = useSearchParams(); const ref = query.get("ref") ?? ""; const path = query.get("path") ?? ""; const page = query.get("page") ?? "1";
  const [items, setItems] = useState<Commit[] | null>(null); const [next, setNext] = useState<number | null>(null); const [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController(); setItems(null); setNext(null); setError("");
    const params = new URLSearchParams({ ref, path, page });
    request<{ items: Commit[]; next_page: number | null }>(`/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/commits?${params}`, { signal: controller.signal }).then(result => { if (!controller.signal.aborted) { setItems(result.items); setNext(result.next_page); } }).catch(failure => {
      if (controller.signal.aborted) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "History unavailable.");
    }); return () => controller.abort();
  }, [owner, repo, ref, path, page, session]);
  return <section><h1>History: {owner}/{repo}</h1><Link to={`${route}?${new URLSearchParams({ ref, path })}`}>Back to files</Link><p>{ref || "Default branch"} {path && `— ${path}`}</p>
    {error && <Alert isInline variant="danger" title={error} />}{!items && !error && <Spinner aria-label="Loading commits" />}
    {items?.length === 0 && <p>No commits on this page.</p>}
    <ol>{items?.map(commit => <li key={commit.sha}><Link to={`${route}/commits/${commit.sha}`}>{commit.commit.message}</Link>{path && /^[0-9a-f]{40}([0-9a-f]{24})?$/i.test(commit.sha) && <> — <a href={`${session.forgejo_url}/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/blame/commit/${commit.sha}/${path.split("/").map(encodeURIComponent).join("/")}`}>Blame at this commit in Forgejo</a></>}<p>{commit.commit.author.name} — {commit.created || commit.commit.author.date} — <code>{commit.sha}</code></p></li>)}</ol>
    {path && <p>Line attribution currently uses Forgejo's native blame page and its own login. Custom blame API coverage remains pending disposition.</p>}
    <Button variant="secondary" isDisabled={!items || Number(page) <= 1} onClick={() => setQuery({ ref, path, page: String(Number(page) - 1) })}>Previous page</Button><Button variant="secondary" isDisabled={next === null} onClick={() => setQuery({ ref, path, page: String(next) })}>Next page</Button>
  </section>;
}
export function CommitDetail({ session }: { session: Session }) {
  const { owner = "", repo = "", sha = "" } = useParams(); const route = repositoryPath(owner, repo);
  const [commit, setCommit] = useState<Commit | null>(null); const [diff, setDiff] = useState<string | null>(null); const [error, setError] = useState(""); const [diffError, setDiffError] = useState("");
  useEffect(() => {
    const controller = new AbortController(); setCommit(null); setDiff(null); setError(""); setDiffError("");
    const api = `/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/commits/${encodeURIComponent(sha)}`;
    request<Commit>(api, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) setCommit(value); }).catch(failure => {
      if (controller.signal.aborted) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Commit unavailable.");
    });
    request<{ text: string }>(`${api}/diff`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) setDiff(value.text); }).catch(failure => {
      if (controller.signal.aborted) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setDiffError(failure instanceof Error ? failure.message : "Diff unavailable.");
    });
    return () => controller.abort();
  }, [owner, repo, sha, session]);
  return <section><h1>Commit <code>{sha}</code></h1><Link to={`${route}/history`}>History</Link>
    {error && <Alert isInline variant="danger" title={error} />}{!commit && !error && <Spinner aria-label="Loading commit" />}
    {commit && <><h2>{commit.commit.message}</h2><p>{commit.commit.author.name} — {commit.created}</p><ul>{commit.files?.map(file => <li key={file.filename}><Link to={`${route}?${new URLSearchParams({ ref: sha, path: file.filename })}`}>{file.filename}</Link> — {file.status}</li>)}</ul></>}
    <h2>Unified diff</h2>{diffError && <Alert isInline variant="warning" title={diffError} />}{diff !== null && <pre tabIndex={0} aria-label="Unified commit diff">{diff || "No textual changes."}</pre>}
    <p>Diffs are bounded to 1 MiB and shown as inert native unified text. Use ordinary Git for larger/binary changes.</p>
  </section>;
}
interface RefEntry { name: string; protected?: boolean; user_can_push?: boolean; message?: string; commit?: { sha: string } }
export function Refs({ session }: { session: Session }) {
  const { owner = "", repo = "" } = useParams(); const route = repositoryPath(owner, repo);
  const [query, setQuery] = useSearchParams(); const kind = query.get("kind") === "tags" ? "tags" : "branches"; const page = query.get("page") ?? "1";
  const [items, setItems] = useState<RefEntry[] | null>(null); const [next, setNext] = useState<number | null>(null);
  const [name, setName] = useState(""); const [source, setSource] = useState(""); const [message, setMessage] = useState(""); const [error, setError] = useState(""); const [busy, setBusy] = useState(false); const [attempt, setAttempt] = useState(0);
  const api = `/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/${kind}`;
  const targetRevision = useRef(0);
  useEffect(() => {
    targetRevision.current++; setBusy(false); setName(""); setSource(""); setMessage("");
    return () => { targetRevision.current++; };
  }, [api, session]);
  useEffect(() => {
    const controller = new AbortController(); setItems(null); setNext(null); setError("");
    request<{ items: RefEntry[]; next_page: number | null }>(`${api}?page=${encodeURIComponent(page)}`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) { setItems(value.items); setNext(value.next_page); } }).catch(failure => {
      if (controller.signal.aborted) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Refs unavailable.");
    }); return () => controller.abort();
  }, [api, page, attempt, session]);
  async function create() {
    if (busy) return; const revision = targetRevision.current; setBusy(true); setError("");
    try {
      await request(api, { method: "POST", csrf: session.csrf_token, body: kind === "branches" ? { name, ref: source } : { name, target: source, message } });
      if (revision === targetRevision.current && useSession.getState().session === session) { setName(""); setAttempt(value => value + 1); }
    } catch (failure) {
      if (revision !== targetRevision.current || useSession.getState().session !== session) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Ref creation was not confirmed. Reload before retrying.");
    } finally { if (revision === targetRevision.current) setBusy(false); }
  }
  return <section aria-busy={busy}><h1>{kind === "branches" ? "Branches" : "Tags"}: {owner}/{repo}</h1><Link to={route}>Files</Link><p><Button variant="link" isDisabled={busy} onClick={() => setQuery({ kind: kind === "branches" ? "tags" : "branches" })}>Show {kind === "branches" ? "tags" : "branches"}</Button></p>
    {error && <Alert isInline variant="danger" title={error} />}{!items && !error && <Spinner aria-label="Loading refs" />}<ul>{items?.map(item => <li key={item.name}><Link to={`${route}?${new URLSearchParams({ ref: item.name })}`}>{item.name}</Link> {item.protected && "Protected"} {item.message}</li>)}</ul>
    <Button variant="secondary" isDisabled={busy || !items || Number(page) <= 1} onClick={() => setQuery({ kind, page: String(Number(page) - 1) })}>Previous page</Button><Button variant="secondary" isDisabled={busy || next === null} onClick={() => setQuery({ kind, page: String(next) })}>Next page</Button>
    <Form onSubmit={event => { event.preventDefault(); void create(); }}><FormGroup label="New name" fieldId="ref-name" isRequired><TextInput id="ref-name" isRequired value={name} isDisabled={busy} onChange={(_, value) => setName(value)} /></FormGroup><FormGroup label="Source branch, tag or commit" fieldId="ref-source" isRequired><TextInput id="ref-source" isRequired value={source} isDisabled={busy} onChange={(_, value) => setSource(value)} /></FormGroup>
      {kind === "tags" && <FormGroup label="Tag message" fieldId="tag-message"><TextArea id="tag-message" value={message} isDisabled={busy} onChange={(_, value) => setMessage(value)} /></FormGroup>}
      <Button type="submit" isDisabled={busy || !name || !source}>Create {kind === "tags" ? "tag" : "branch"}</Button>
    </Form><p>Forgejo enforces native permissions and protection rules. No environment state is changed.</p>
  </section>;
}
export function Compare({ session }: { session: Session }) {
  const { owner = "", repo = "" } = useParams(); const route = repositoryPath(owner, repo);
  const [query, setQuery] = useSearchParams(); const base = query.get("base") ?? ""; const head = query.get("head") ?? "";
  const [baseDraft, setBase] = useState(base); const [headDraft, setHead] = useState(head);
  const [result, setResult] = useState<{ commits: Commit[]; files: { filename: string; status: string }[]; total_commits: string; base_sha: string; head_sha: string } | null>(null); const [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController(); setResult(null); setError(""); setBase(base); setHead(head);
    if (base && head) request<typeof result>(`/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/compare?${new URLSearchParams({ base, head })}`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) setResult(value); }).catch(failure => {
      if (controller.signal.aborted) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Comparison unavailable.");
    }); return () => controller.abort();
  }, [owner, repo, base, head, session]);
  return <section><h1>Compare: {owner}/{repo}</h1><Link to={route}>Files</Link><Form onSubmit={event => { event.preventDefault(); setQuery({ base: baseDraft, head: headDraft }); }}>
    <FormGroup label="Base ref" fieldId="compare-base"><TextInput id="compare-base" value={baseDraft} onChange={(_, value) => setBase(value)} /></FormGroup><FormGroup label="Head ref" fieldId="compare-head"><TextInput id="compare-head" value={headDraft} onChange={(_, value) => setHead(value)} /></FormGroup><Button type="submit" isDisabled={!baseDraft || !headDraft}>Compare refs</Button></Form>
    {error && <Alert isInline variant="danger" title={error} />}{base && head && !result && !error && <Spinner aria-label="Comparing refs" />}
    {result && <><h2>{result.total_commits} commits reported</h2><ol>{result.commits?.map(commit => <li key={commit.sha}><Link to={`${route}/commits/${commit.sha}`}>{commit.commit.message}</Link></li>)}</ol><p>Compared snapshots: <code>{result.base_sha}</code> … <code>{result.head_sha}</code></p><h2>Files touched by the reported commits</h2><ul>{result.files?.map((file, index) => <li key={`${index}:${file.filename}`}>{file.filename} — {file.status}</li>)}</ul><p>Forgejo returns per-commit file entries here; paths may repeat and reverted changes may appear. This is not a net changed-file list. Open each pinned commit for its bounded unified diff.</p><a href={`${session.forgejo_url}/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/compare/${result.base_sha}...${result.head_sha}`}>Aggregate comparison in Forgejo</a><p>Aggregate diff currently requires the native Forgejo view and its own login. This API coverage gap remains pending disposition; no environment is merged or altered.</p></>}
  </section>;
}
