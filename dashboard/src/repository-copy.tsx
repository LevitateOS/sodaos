import { useEffect, useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { Alert, Button, Checkbox, Form, FormGroup, TextInput } from "@patternfly/react-core";
import { APIError, request, type Session } from "./api";
import { useSession } from "./session";
import { repositoryPath, type Repository } from "./repositories";

export function ForkRepository({ session }: { session: Session }) {
  const { owner = "", repo = "" } = useParams(); const navigate = useNavigate(); const targetRevision = useRef(0);
  const [name, setName] = useState(repo); const [busy, setBusy] = useState(false); const [error, setError] = useState("");
  useEffect(() => {
    targetRevision.current++; setName(repo); setBusy(false); setError("");
    return () => { targetRevision.current++; };
  }, [owner, repo, session]);
  async function fork() {
    if (busy) return; const revision = targetRevision.current; setBusy(true); setError("");
    try {
      const result = await request<Repository>(`/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/fork`, { method: "POST", csrf: session.csrf_token, body: { name } });
      if (revision === targetRevision.current && useSession.getState().session === session) navigate(repositoryPath(result.owner.login, result.name));
    } catch (failure) {
      if (revision !== targetRevision.current || useSession.getState().session !== session) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Fork not confirmed; inspect repositories before retrying.");
    } finally { if (revision === targetRevision.current) setBusy(false); }
  }
  return <section aria-busy={busy}><h1>Fork {owner}/{repo}</h1><Link to={repositoryPath(owner, repo)}>Back to repository</Link><p>Forgejo creates a native fork for your account. No development environment, Linux account or service state is copied.</p>{error && <Alert isInline variant="danger" title={error} />}
    <Form onSubmit={event => { event.preventDefault(); void fork(); }}><FormGroup label="Fork repository name" fieldId="fork-name"><TextInput id="fork-name" value={name} onChange={(_, value) => setName(value)} isDisabled={busy} maxLength={100} /></FormGroup><Button type="submit" isDisabled={busy} isLoading={busy}>Fork in Forgejo</Button></Form>
  </section>;
}
export function ImportRepository({ session }: { session: Session }) {
  const navigate = useNavigate(); const targetRevision = useRef(0); const formRef = useRef<HTMLFormElement>(null);
  const [privateRepo, setPrivate] = useState(true); const [busy, setBusy] = useState(false); const [error, setError] = useState("");
  useEffect(() => {
    targetRevision.current++; formRef.current?.reset(); setPrivate(true); setBusy(false); setError("");
    return () => { targetRevision.current++; };
  }, [session]);
  async function submit(form: HTMLFormElement) {
    if (busy) return; const revision = targetRevision.current; const data = new FormData(form);
    const body = { url: String(data.get("url") ?? ""), name: String(data.get("name") ?? ""), private: privateRepo, username: String(data.get("username") ?? ""), password: String(data.get("password") ?? ""), token: String(data.get("token") ?? "") };
    // Clear credential inputs immediately, but retain non-secret source/name
    // fields for inspection after a native failure. Never persist this form.
    for (const field of ["username", "password", "token"]) {
      const input = form.elements.namedItem(field); if (input instanceof HTMLInputElement) input.value = "";
      data.delete(field);
    }
    setBusy(true); setError("");
    try {
      const result = await request<Repository>("/api/forgejo/repositories/import", { method: "POST", csrf: session.csrf_token, body });
      if (revision === targetRevision.current && useSession.getState().session === session) navigate(repositoryPath(result.owner.login, result.name));
    } catch (failure) {
      if (revision !== targetRevision.current || useSession.getState().session !== session) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Import not confirmed. Inspect native repositories before retrying.");
    } finally { body.username = ""; body.password = ""; body.token = ""; if (revision === targetRevision.current) setBusy(false); }
  }
  return <section aria-busy={busy}><h1>Import a Git repository</h1><Link to="/repositories">Repositories</Link><p>Forgejo performs a one-time HTTPS Git import into your account under its native permissions and network policy. This does not execute repository code, create an environment, configure a mirror or import provider-specific issues.</p>
    {error && <Alert isInline variant="danger" title={error} />}
    <Form ref={formRef} onSubmit={event => { event.preventDefault(); void submit(event.currentTarget); }}>
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
