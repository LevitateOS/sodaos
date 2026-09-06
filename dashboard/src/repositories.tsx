import { useEffect, useRef, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { Alert, Button, Checkbox, Form, FormGroup, Spinner, TextArea, TextInput } from "@patternfly/react-core";
import { APIError, request, type Session } from "./api";
import { useSession } from "./session";
import { Markdown } from "./markdown";

export interface Repository {
  id: string; name: string; full_name: string;
  owner: { id: string; login: string }; description: string; private: boolean;
  default_branch: string; clone_url: string; ssh_url: string; empty: boolean;
}
interface RepositoryPage { items: Repository[]; next_page: number | null }
interface Content { name: string; path: string; type: string; sha: string; size: string; text: string | null; unavailable: boolean }
export function repositoryPath(owner: string, name: string) { return `/repositories/${encodeURIComponent(owner)}/${encodeURIComponent(name)}`; }
export function RepositoryList({ session }: { session: Session }) {
  const [search, setSearch] = useSearchParams();
  const page = search.get("page") ?? "1";
  const [result, setResult] = useState<RepositoryPage | null>(null);
  const [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController(); setResult(null); setError("");
    request<RepositoryPage>(`/api/forgejo/repositories?page=${encodeURIComponent(page)}`, { signal: controller.signal }).then(data => {
      if (!controller.signal.aborted) setResult(data);
    }).catch(failure => {
      if (controller.signal.aborted) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Could not list repositories.");
    });
    return () => controller.abort();
  }, [page, session]);
  return <section><h1>Repositories</h1><p>Forgejo determines which repositories you can see, including collaborator access. Creating a repository does not create or join an environment.</p>
    <Link to="/repositories/new">Create repository</Link>
    {error && <Alert isInline variant="danger" title={error} />}
    {!result && !error && <Spinner aria-label="Loading repositories" />}
    {result && <><ul>{result.items.map(repo => <li key={repo.id}><Link to={repositoryPath(repo.owner.login, repo.name)}>{repo.full_name}</Link> {repo.private ? "Private" : "Public"}<p>{repo.description}</p></li>)}</ul>
      {result.items.length === 0 && <p>No repositories on this page.</p>}
      <Button variant="secondary" isDisabled={Number(page) <= 1} onClick={() => setSearch({ page: String(Number(page) - 1) })}>Previous page</Button>
      <Button variant="secondary" isDisabled={result.next_page === null} onClick={() => setSearch({ page: String(result.next_page) })}>Next page</Button>
    </>}
  </section>;
}
export function CreateRepository({ session }: { session: Session }) {
  const navigate = useNavigate();
  const active = useRef(true);
  useEffect(() => { active.current = true; return () => { active.current = false; }; }, []);
  const [name, setName] = useState(""); const [description, setDescription] = useState("");
  const [privateRepo, setPrivate] = useState(true); const [initialize, setInitialize] = useState(true);
  const [busy, setBusy] = useState(false); const [error, setError] = useState("");
  async function submit() {
    if (busy) return; setBusy(true); setError("");
    try {
      const repo = await request<Repository>("/api/forgejo/repositories", { method: "POST", csrf: session.csrf_token, body: { name, description, private: privateRepo, auto_init: initialize } });
      if (!active.current || useSession.getState().session !== session) return;
      navigate(repositoryPath(repo.owner.login, repo.name));
    } catch (failure) {
      if (useSession.getState().session !== session) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof APIError ? failure.message : "Creation was not confirmed. Inspect repositories before retrying; it may have completed.");
    } finally { setBusy(false); }
  }
  return <section aria-busy={busy}><h1>Create repository</h1>{error && <Alert isInline variant="danger" title={error} />}
    <Form onSubmit={event => { event.preventDefault(); void submit(); }}>
      <FormGroup label="Repository name" fieldId="repository-name" isRequired><TextInput id="repository-name" value={name} onChange={(_, value) => setName(value)} isRequired isDisabled={busy} maxLength={100} /></FormGroup>
      <FormGroup label="Description" fieldId="repository-description"><TextArea id="repository-description" value={description} onChange={(_, value) => setDescription(value)} isDisabled={busy} maxLength={2048} /></FormGroup>
      <Checkbox id="repository-private" label="Private repository" isChecked={privateRepo} onChange={(_, value) => setPrivate(value)} isDisabled={busy} />
      <Checkbox id="repository-initialize" label="Initialize repository with a README" isChecked={initialize} onChange={(_, value) => setInitialize(value)} isDisabled={busy} />
      <Button type="submit" isDisabled={busy || !name.trim()} isLoading={busy}>Create repository in Forgejo</Button>
    </Form>
  </section>;
}
export function RepositoryDetail({ session }: { session: Session }) {
  const active = useRef(true);
  useEffect(() => { active.current = true; return () => { active.current = false; }; }, []);
  const { owner = "", repo = "" } = useParams();
  const [search, setSearch] = useSearchParams(); const ref = search.get("ref") ?? ""; const path = search.get("path") ?? "";
  const [repository, setRepository] = useState<Repository | null>(null);
  const [contents, setContents] = useState<Content[] | null>(null);
  const [readme, setReadme] = useState<Content | null>(null);
  const [readmeError, setReadmeError] = useState("");
  const [error, setError] = useState(""); const [contentError, setContentError] = useState("");
  const [refDraft, setRefDraft] = useState(ref); const [busy, setBusy] = useState(false);
  const navigate = useNavigate();
  const api = `/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}`;
  useEffect(() => {
    const controller = new AbortController(); setRepository(null); setError("");
    request<Repository>(api, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) setRepository(value); }).catch(failure => {
      if (controller.signal.aborted) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Repository unavailable.");
    });
    return () => controller.abort();
  }, [api, session]);
  useEffect(() => {
    const controller = new AbortController(); setContents(null); setReadme(null); setReadmeError(""); setContentError(""); setRefDraft(ref);
    if (!repository || repository.empty) return () => controller.abort();
    const query = new URLSearchParams({ ref: ref || repository.default_branch, path });
    request<{ items: Content[] }>(`${api}/contents?${query}`, { signal: controller.signal }).then(async value => {
      if (controller.signal.aborted) return;
      setContents(value.items);
      const entry = path === "" ? value.items.find(item => item.type === "file" && /^readme(?:\.md|\.markdown)?$/i.test(item.name)) : undefined;
      if (entry) {
        try {
          const readmeQuery = new URLSearchParams({ ref: ref || repository.default_branch, path: entry.path });
          const result = await request<{ items: Content[] }>(`${api}/contents?${readmeQuery}`, { signal: controller.signal });
          if (!controller.signal.aborted) setReadme(result.items[0] ?? null);
        } catch (failure) { if (!controller.signal.aborted) setReadmeError(failure instanceof Error ? failure.message : "README unavailable."); }
      }
    }).catch(failure => {
      if (controller.signal.aborted) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setContentError(failure instanceof Error ? failure.message : "Files unavailable.");
    });
    return () => controller.abort();
  }, [api, repository, ref, path, session]);
  async function createEnvironment() {
    if (!repository || busy) return; setBusy(true); setError("");
    try {
      const result = await request<{ id: string }>("/api/environments", { method: "POST", csrf: session.csrf_token, body: { owner: repository.owner.login, repository: repository.name } });
      if (active.current && useSession.getState().session === session) navigate(`/environments/${result.id}`);
    } catch (failure) {
      if (useSession.getState().session !== session) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Creation not confirmed. Inspect the environment list before retrying.");
    } finally { setBusy(false); }
  }
  return <section aria-busy={busy}><h1>{owner}/{repo}</h1>{error && <Alert isInline variant="danger" title={error} />}
    {!repository && !error && <Spinner aria-label="Loading repository" />}
    {repository && <><p>{repository.description}</p><h2>Personal Git access</h2><p>Use your own Forgejo Git credentials; joining an environment grants no Git permissions.</p><pre>{repository.clone_url}{"\n"}{repository.ssh_url}</pre>
      <h2>Development environment</h2><Link to="/environments">Discover existing environments and incomplete reservations</Link>
      {repository.owner.id === session.user.id && <Button isDisabled={busy} isLoading={busy} onClick={() => void createEnvironment()}>Create persistent Rocky environment</Button>}
      <p>The human owner administers the environment. Creation does not join you automatically.</p>
      <h2>Files</h2>{repository.empty ? <p>This repository is empty. Push with ordinary Git to add files.</p> : <>
        <Form onSubmit={event => { event.preventDefault(); setSearch({ ref: refDraft, path: "" }); }}><FormGroup label="Branch, tag or commit" fieldId="repository-ref"><TextInput id="repository-ref" value={refDraft} placeholder={repository.default_branch} onChange={(_, value) => setRefDraft(value)} /></FormGroup><Button type="submit">Browse ref</Button></Form>
        {path && <Button variant="link" onClick={() => setSearch({ ref, path: path.split("/").slice(0, -1).join("/") })}>Parent directory</Button>}
        {contentError && <Alert isInline variant="danger" title={contentError} />}
        {!contents && !contentError && <Spinner aria-label="Loading files" />}
        <ul>{contents?.map(item => <li key={item.path}><Button variant="link" onClick={() => setSearch({ ref, path: item.path })}>{item.path}</Button> {item.type} ({item.size} bytes)
          {item.text !== null && (/\.(md|markdown)$/i.test(item.name) ? <Markdown text={item.text} repository={{ route: repositoryPath(owner, repo), ref: ref || repository.default_branch, path: item.path }} /> : <pre tabIndex={0} aria-label={`Contents of ${item.path}`}>{item.text}</pre>)}
          {item.type === "file" && <p><a href={`${api}/download?${new URLSearchParams({ ref: ref || repository.default_branch, path: item.path })}`}>Download file (up to 8 MiB)</a></p>}
          {item.unavailable && <p>Binary, oversized or unavailable content is not rendered. Use the bounded download or your native Git checkout.</p>}
        </li>)}</ul>
        {readmeError && <Alert isInline variant="warning" title={`README: ${readmeError}`} />}
        {readme && <><h2>{readme.name}</h2>{readme.text !== null ? <Markdown text={readme.text} repository={{ route: repositoryPath(owner, repo), ref: ref || repository.default_branch, path: readme.path }} /> : <p>README content is oversized, binary or unavailable.</p>}</>}
        <p>Markdown uses standard GFM without native Forgejo extensions. Raw HTML is ignored and images are not fetched automatically.</p>
      </>}
    </>}
  </section>;
}
