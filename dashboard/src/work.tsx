import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { Alert, Button, Form, FormGroup, FormSelect, FormSelectOption, Spinner, TextInput } from "@patternfly/react-core";
import { APIError, request, type Session } from "./api";
import { useSession } from "./session";
import { repositoryPath } from "./repositories";
interface WorkItem { id: string; number: string; title: string; state: string; kind: string; repository: { name: string; owner: string; full_name: string } }
interface Repository { id: string; name: string; full_name: string; owner: { login: string } }
export function WorkResults({ session, query, repositories = false }: { session: Session; query: string; repositories?: boolean }) {
  const [items, setItems] = useState<WorkItem[] | null>(null); const [repos, setRepos] = useState<Repository[] | null>(null); const [next, setNext] = useState<number | null>(null); const [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController(); setItems(null); setRepos(null); setNext(null); setError("");
    function failed(failure: unknown) { if (controller.signal.aborted) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Native search unavailable."); }
    if (repositories) request<{ items: Repository[]; next_page: number | null }>(`/api/forgejo/repository-search?${query}`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) { setRepos(value.items); setNext(value.next_page); } }).catch(failed);
    else request<{ items: WorkItem[]; next_page: number | null }>(`/api/forgejo/work?${query}`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) { setItems(value.items); setNext(value.next_page); } }).catch(failed);
    return () => controller.abort();
  }, [query, repositories, session]);
  const params = new URLSearchParams(query); const page = Number(params.get("page") ?? "1");
  function pageLink(page: number) { const nextQuery = new URLSearchParams(query); nextQuery.set("page", String(page)); if (repositories) nextQuery.set("type", "repositories"); return `/search?${nextQuery}`; }
  return <section>{error && <Alert isInline variant="warning" title={error} />}{!items && !repos && !error && <Spinner aria-label="Searching native visible work" />}
    {items && <ul>{items.map(item => <li key={item.id}><Link to={`${repositoryPath(item.repository.owner, item.repository.name)}/${item.kind === "pulls" ? "pulls" : "issues"}/${item.number}`}>{item.repository.full_name} #{item.number}: {item.title}</Link> — {item.state}</li>)}</ul>}
    {repos && <ul>{repos.map(repo => <li key={repo.id}><Link to={repositoryPath(repo.owner.login, repo.name)}>{repo.full_name}</Link></li>)}</ul>}
    {(items?.length === 0 || repos?.length === 0) && <p>No matching visible results on this page.</p>}{page > 1 && <Link to={pageLink(page - 1)}>Previous results</Link>}{next !== null && <Link to={pageLink(next)}>Next results</Link>}
  </section>;
}
export function Search({ session }: { session: Session }) {
  const [params, setParams] = useSearchParams(); const encoded = params.toString(); const [search, setSearch] = useState(params.get("q") ?? ""); const [owner, setOwner] = useState(params.get("owner") ?? ""); const [kind, setKind] = useState(params.get("type") ?? "issues"); const [state, setState] = useState(params.get("state") ?? "open"); const [personal, setPersonal] = useState("");
  useEffect(() => { const value = new URLSearchParams(encoded); setSearch(value.get("q") ?? ""); setOwner(value.get("owner") ?? ""); setKind(value.get("type") ?? "issues"); setState(value.get("state") ?? "open"); setPersonal(["assigned", "created", "mentioned", "review_requested", "reviewed"].find(key => value.get(key) === "true") ?? ""); }, [encoded]);
  const nativeQuery = new URLSearchParams(params); const repositories = params.get("type") === "repositories"; if (repositories) nativeQuery.delete("type");
  return <section><h1>Search native visible work</h1><Form onSubmit={event => { event.preventDefault(); setParams({ q: search, owner, type: kind, state, ...(personal && kind !== "repositories" ? { [personal]: "true" } : {}) }); }}>
    <FormGroup label="Search text" fieldId="global-search"><TextInput id="global-search" value={search} onChange={(_, value) => setSearch(value)} /></FormGroup><FormGroup label="Native repository owner (optional)" fieldId="global-owner"><TextInput id="global-owner" value={owner} onChange={(_, value) => setOwner(value)} /></FormGroup>
    <FormGroup label="Search kind" fieldId="global-kind"><FormSelect id="global-kind" value={kind} onChange={(_, value) => setKind(value)}>{["repositories", "issues", "pulls"].map(value => <FormSelectOption key={value} value={value} label={value} />)}</FormSelect></FormGroup>
    {kind !== "repositories" && <><FormGroup label="State" fieldId="global-state"><FormSelect id="global-state" value={state} onChange={(_, value) => setState(value)}>{["open", "closed", "all"].map(value => <FormSelectOption key={value} value={value} label={value} />)}</FormSelect></FormGroup><FormGroup label="My native relationship" fieldId="global-personal"><FormSelect id="global-personal" value={personal} onChange={(_, value) => setPersonal(value)}><FormSelectOption value="" label="Any visible work" />{["assigned", "created", "mentioned", "review_requested", "reviewed"].map(value => <FormSelectOption key={value} value={value} label={value} />)}</FormSelect></FormGroup></>}
    <Button type="submit">Search Forgejo</Button></Form><WorkResults session={session} query={nativeQuery.toString()} repositories={repositories} />
  </section>;
}
export function MyWork({ session }: { session: Session }) {
  const [environments, setEnvironments] = useState<{ id: string; name: string; repository: string }[] | null>(null); const [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController(); request<{ items: { id: string; name: string; repository: string }[] }>("/api/environments", { signal: controller.signal }).then(value => { if (!controller.signal.aborted) setEnvironments(value.items); }).catch(failure => { if (controller.signal.aborted) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Soda environments unavailable."); }); return () => controller.abort();
  }, [session]);
  return <section><h1>My work</h1><p>Welcome, {session.user.soda_display_name || session.user.login}. Forgejo owns the work below; Soda environments are separately sourced.</p>
    <h2>Assigned native issues</h2><WorkResults session={session} query="assigned=true&type=issues&state=open" /><h2>Requested native reviews</h2><WorkResults session={session} query="review_requested=true&type=pulls&state=open" />
    <h2>My owned native repositories</h2><Link to="/repositories">All accessible repositories</Link><WorkResults session={session} repositories query={new URLSearchParams({ owner: session.user.login }).toString()} /><h2>Visible Soda environments</h2>{error && <Alert isInline variant="warning" title={error} />}{!environments && !error && <Spinner aria-label="Loading Soda environments" />}<ul>{environments?.map(item => <li key={item.id}><Link to={`/environments/${encodeURIComponent(item.id)}`}>{item.name}</Link> — {item.repository}</li>)}</ul><Link to="/environments">Environment directory and explicit join</Link>
    <Alert isInline variant="warning" title="Preview: installed developer, direct-routing and workload/persistence verification remains pending." />
  </section>;
}
