import { useEffect, useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { Alert, Button, Checkbox, Form, FormGroup, TextInput } from "@patternfly/react-core";
import { APIError, request, type Session } from "./api";
import { useSession } from "./session";
import { repositoryPath, type Repository } from "./repositories";

export function ForkRepository({ session }: { session: Session }) {
  const { owner = "", repo = "" } = useParams(); const navigate = useNavigate(); const active = useRef(true);
  useEffect(() => { active.current = true; return () => { active.current = false; }; }, []);
  const [name, setName] = useState(repo); const [busy, setBusy] = useState(false); const [error, setError] = useState("");
  async function fork() {
    if (busy) return; setBusy(true); setError("");
    try {
      const result = await request<Repository>(`/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/fork`, { method: "POST", csrf: session.csrf_token, body: { name } });
      if (active.current && useSession.getState().session === session) navigate(repositoryPath(result.owner.login, result.name));
    } catch (failure) {
      if (!active.current || useSession.getState().session !== session) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Fork not confirmed; inspect repositories before retrying.");
    } finally { setBusy(false); }
  }
  return <section aria-busy={busy}><h1>Fork {owner}/{repo}</h1><Link to={repositoryPath(owner, repo)}>Back to repository</Link><p>Forgejo creates a native fork for your account. No development environment, Linux account or service state is copied.</p>{error && <Alert isInline variant="danger" title={error} />}
    <Form onSubmit={event => { event.preventDefault(); void fork(); }}><FormGroup label="Fork repository name" fieldId="fork-name"><TextInput id="fork-name" value={name} onChange={(_, value) => setName(value)} isDisabled={busy} maxLength={100} /></FormGroup><Button type="submit" isDisabled={busy} isLoading={busy}>Fork in Forgejo</Button></Form>
  </section>;
}
export function ImportRepository({ session }: { session: Session }) {
  const navigate = useNavigate(); const active = useRef(true);
  useEffect(() => { active.current = true; return () => { active.current = false; }; }, []);
  const [privateRepo, setPrivate] = useState(true); const [busy, setBusy] = useState(false); const [error, setError] = useState("");
  async function submit(form: HTMLFormElement) {
    if (busy) return; const data = new FormData(form);
    const body = { url: String(data.get("url") ?? ""), name: String(data.get("name") ?? ""), private: privateRepo, username: String(data.get("username") ?? ""), password: String(data.get("password") ?? ""), token: String(data.get("token") ?? "") };
    form.reset(); setBusy(true); setError("");
    try {
      const result = await request<Repository>("/api/forgejo/repositories/import", { method: "POST", csrf: session.csrf_token, body });
      if (active.current && useSession.getState().session === session) navigate(repositoryPath(result.owner.login, result.name));
    } catch (failure) {
      if (!active.current || useSession.getState().session !== session) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Import not confirmed. Inspect native repositories before retrying.");
    } finally { body.password = ""; body.token = ""; data.delete("password"); data.delete("token"); setBusy(false); }
  }
  return <section aria-busy={busy}><h1>Import a Git repository</h1><Link to="/repositories">Repositories</Link><p>Forgejo performs a one-time HTTPS Git import into your account under its native permissions and network policy. This does not execute repository code, create an environment, configure a mirror or import provider-specific issues.</p>
    {error && <Alert isInline variant="danger" title={error} />}
    <Form onSubmit={event => { event.preventDefault(); void submit(event.currentTarget); }}>
      <FormGroup label="HTTPS Git clone URL — no embedded credentials" fieldId="import-url" isRequired><TextInput id="import-url" name="url" type="url" isRequired isDisabled={busy} autoComplete="off" maxLength={4096} /></FormGroup>
      <FormGroup label="New repository name" fieldId="import-name" isRequired><TextInput id="import-name" name="name" isRequired isDisabled={busy} maxLength={100} /></FormGroup>
      <Checkbox id="import-private" label="Private repository" isChecked={privateRepo} isDisabled={busy} onChange={(_, value) => setPrivate(value)} />
      <FormGroup label="Optional source username" fieldId="import-username"><TextInput id="import-username" name="username" isDisabled={busy} autoComplete="off" maxLength={255} /></FormGroup>
      <FormGroup label="Optional source password" fieldId="import-password"><TextInput id="import-password" name="password" type="password" isDisabled={busy} autoComplete="off" maxLength={4096} /></FormGroup>
      <FormGroup label="Optional source token" fieldId="import-token"><TextInput id="import-token" name="token" type="password" isDisabled={busy} autoComplete="off" maxLength={4096} /></FormGroup>
      <Button type="submit" isDisabled={busy} isLoading={busy}>Import through Forgejo</Button>
    </Form><p>Credentials are sent only with this request to Forgejo, never stored by Soda or browser persistence. They are cleared from the form on submission. A lost response is not permission to retry an ambiguous import automatically.</p>
  </section>;
}
