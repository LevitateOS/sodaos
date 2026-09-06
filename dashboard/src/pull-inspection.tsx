import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { Alert, Button, Spinner } from "@patternfly/react-core";
import { APIError, request, type Session } from "./api";
import { useSession } from "./session";
import { repositoryPath } from "./repositories";
import type { Pull } from "./pulls";
interface Commit { sha: string; commit: { message: string; author: { name: string; date: string } } }
interface ChangedFile { filename: string; previous_filename: string; status: string; additions: string; deletions: string }
interface Status { id: string; context: string; description: string; state: string }
export function PullInspection({ session, owner, repo, pull, tab }: { session: Session; owner: string; repo: string; pull: Pull; tab: "commits" | "files" | "checks" }) {
  const [query, setQuery] = useSearchParams(); const page = query.get("page") ?? "1"; const api = `/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}`;
  const [commits, setCommits] = useState<Commit[] | null>(null); const [files, setFiles] = useState<ChangedFile[] | null>(null); const [statuses, setStatuses] = useState<Status[] | null>(null); const [state, setState] = useState(""); const [diff, setDiff] = useState<string | null>(null);
  const [next, setNext] = useState<number | null>(null); const [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController(); setCommits(null); setFiles(null); setStatuses(null); setDiff(null); setError(""); setNext(null);
    function failed(failure: unknown) { if (controller.signal.aborted) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Native inspection unavailable."); }
    if (tab === "commits") {
      request<{ items: Commit[]; next_page: number | null }>(`${api}/pulls/${pull.number}/commits?page=${encodeURIComponent(page)}`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) { setCommits(value.items); setNext(value.next_page); } }).catch(failed);
    } else if (tab === "checks") {
      request<{ items: Status[]; state: string; next_page: number | null }>(`${api}/commit-status/${pull.head.sha}?page=${encodeURIComponent(page)}`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) { setStatuses(value.items); setState(value.state); setNext(value.next_page); } }).catch(failed);
    } else {
      const snapshot = new URLSearchParams({ head: pull.head.sha, base: pull.base.sha, merge_base: pull.merge_base, page });
      Promise.all([request<{ items: ChangedFile[]; next_page: number | null }>(`${api}/pulls/${pull.number}/files?${snapshot}`, { signal: controller.signal }), request<{ text: string }>(`${api}/pulls/${pull.number}/diff?${snapshot}`, { signal: controller.signal })]).then(([listing, patch]) => { if (!controller.signal.aborted) { setFiles(listing.items); setNext(listing.next_page); setDiff(patch.text); } }).catch(failed);
    } return () => controller.abort();
  }, [api, pull, tab, page, session]);
  function turn(page: number) { const params = new URLSearchParams(query); params.set("page", String(page)); setQuery(params); }
  return <section><h2>{tab}</h2>{error && <Alert isInline variant="danger" title={error} />}{!commits && !files && !statuses && !error && <Spinner aria-label="Loading pull request inspection" />}
    {commits && <ul>{commits.map(commit => <li key={commit.sha}><Link to={`${repositoryPath(owner, repo)}/commits/${commit.sha}`}>{commit.sha.slice(0, 12)} {commit.commit.message}</Link> — {commit.commit.author.name}</li>)}</ul>}
    {files && <><p>Changed files and inert unified diff checked against the displayed head, base and merge base. Binary, oversized and unsupported encodings require native Git; no truncated diff is presented as complete.</p><ul>{files.map(file => <li key={file.filename}>{file.status}: {file.filename}{file.previous_filename && ` (from ${file.previous_filename})`} +{file.additions}/−{file.deletions}</li>)}</ul><pre className="soda-file-text">{diff}</pre></>}
    {statuses && <><p>Native base-repository statuses for head {pull.head.sha}: {state}. Forgejo remains the authority for all merge protection and review checks; this is not a Soda CI scheduler.</p><ul>{statuses.map(status => <li key={status.id}>{status.context}: {status.state} — {status.description}</li>)}</ul>{statuses.length === 0 && <p>No statuses on this native page. This does not prove all merge requirements pass.</p>}</>}
    <Button variant="secondary" isDisabled={Number(page) <= 1} onClick={() => turn(Number(page) - 1)}>Previous page</Button><Button variant="secondary" isDisabled={next === null} onClick={() => next !== null && turn(next)}>Next page</Button>
  </section>;
}
