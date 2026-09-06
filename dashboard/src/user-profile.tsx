import { useEffect, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { Alert, Button, Form, FormGroup, Spinner, TextInput } from "@patternfly/react-core";
import { APIError, request, type Session } from "./api";
import { useSession } from "./session";
import { WorkResults } from "./work";
import { repositoryPath } from "./repositories";
interface Profile { user: { id: string; login: string; full_name: string }; description: string; location: string; pronouns: string; created: string }
interface Activity { id: string; actor: { login: string }; repository: { name: string; full_name: string; owner: { login: string } } | null; operation: string; created: string; content: string; ref: string }
function UserActivity({ session, login }: { session: Session; login: string }) {
  const [query, setQuery] = useSearchParams(); const page = query.get("page") ?? "1"; const date = query.get("date") ?? ""; const [inputDate, setInputDate] = useState(date); const [items, setItems] = useState<Activity[] | null>(null); const [next, setNext] = useState<number | null>(null); const [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController(); setInputDate(date); setItems(null); setError(""); setNext(null);
    request<{ items: Activity[]; next_page: number | null }>(`/api/forgejo/users/${encodeURIComponent(login)}/activity?${new URLSearchParams({ page, date })}`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) { setItems(value.items); setNext(value.next_page); } }).catch(failure => { if (controller.signal.aborted) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Native activity unavailable."); }); return () => controller.abort();
  }, [login, page, date, session]);
  return <section><h2>Visible native activity</h2><Form onSubmit={event => { event.preventDefault(); setQuery({ date: inputDate }); }}><FormGroup label="Activity date (optional)" fieldId="activity-date"><TextInput id="activity-date" type="date" value={inputDate} onChange={(_, value) => setInputDate(value)} /></FormGroup><Button type="submit">Filter native activity</Button></Form>{error && <Alert isInline variant="warning" title={error} />}{!items && !error && <Spinner aria-label="Loading native activity" />}<ul>{items?.map(item => <li key={item.id}>{item.created} — {item.actor.login} — {item.operation} — {item.repository && <Link to={repositoryPath(item.repository.owner.login, item.repository.name)}>{item.repository.full_name}</Link>} {item.ref}{item.content && <details><summary>Native activity content (inert text)</summary><pre className="soda-file-text">{item.content}</pre></details>}</li>)}</ul>{items?.length === 0 && <p>No visible native activity on this page.</p>}<Button variant="secondary" isDisabled={Number(page) <= 1} onClick={() => setQuery({ date, page: String(Number(page) - 1) })}>Previous page</Button><Button variant="secondary" isDisabled={next === null} onClick={() => next !== null && setQuery({ date, page: String(next) })}>Next page</Button></section>;
}
export function UserProfile({ session }: { session: Session }) {
  const { login = "" } = useParams(); const [profile, setProfile] = useState<Profile | null>(null); const [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController(); setProfile(null); setError(""); request<Profile>(`/api/forgejo/users/${encodeURIComponent(login)}`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) setProfile(value); }).catch(failure => { if (controller.signal.aborted) return; if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session); else setError(failure instanceof Error ? failure.message : "Native profile unavailable."); }); return () => controller.abort();
  }, [login, session]);
  return <section><h1>Forgejo profile: {login}</h1>{error && <Alert isInline variant="danger" title={error} />}{!profile && !error && <Spinner aria-label="Loading native profile" />}{profile && <><p>{profile.user.full_name} — {profile.pronouns}</p><p>{profile.description}</p><p>{profile.location} — joined {profile.created}</p><h2>Visible native repositories</h2><WorkResults session={session} repositories query={new URLSearchParams({ owner: login }).toString()} /><UserActivity session={session} login={login} /></>}</section>;
}
