import { StrictMode, useEffect } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, Link, Route, Routes } from "react-router-dom";
import { Alert, Button, Page, PageSection, Spinner } from "@patternfly/react-core";
import "@patternfly/react-core/dist/styles/base.css";
import "./style.css";
import logo from "../../assets/branding/source/soda-logo-horizontal.svg";
import { useSession } from "./session";
import { Profile } from "./profile";

export function App() {
  const { phase, session, error, load, logout } = useSession();
  useEffect(() => { void load(); }, [load]);
  return <Page sidebar={null} mainAriaLabel="Soda" className="soda-preview">
    <PageSection>
      <header><Link to="/"><img src={logo} alt="Soda" /></Link><span>React preview</span></header>
      {(phase === "loading" || phase === "signing-out") && <Spinner aria-label={phase === "loading" ? "Reading session" : "Signing out"} />}
      {phase === "error" && <><Alert isInline variant="danger" title={error} /><Button onClick={() => void load()}>Check session</Button></>}
      {phase === "anonymous" && <><h1>Welcome to Soda</h1><p>Sign in with Forgejo. Your password stays with Forgejo.</p><a href="/login?return_to=%2Fapp%2F">Sign in with Forgejo</a></>}
      {phase === "authenticated" && session && <>
        <nav aria-label="Soda navigation">
          <Link to="/">Overview</Link><Link to="/profile">Profile and development keys</Link><Link to="/help">Help</Link>
          <a href="/projects">Projects (existing dashboard)</a>
          {session.soda_operator && <a href="/people">People (existing dashboard)</a>}
          <a href={session.forgejo_url}>Open Forgejo</a>
          <Button variant="link" onClick={() => void logout()}>Sign out of Soda</Button>
        </nav>
        <Routes>
          <Route path="/" element={<><h1>Welcome, {session.user.soda_display_name || session.user.login}</h1><p>This preview uses your real Soda session. Profile preferences and development-key registration are connected; repository and administrator migration is still in progress.</p><p><a href="/projects">Continue to your existing projects</a></p></>} />
          <Route path="/profile" element={<Profile key={session.user.id} session={session} />} />
          <Route path="/help" element={<><h1>Development access</h1><p>Register only public SSH keys. Adding a key does not rotate keys already installed in environments. Explicitly join through Projects, then use the displayed project IP with SSH, SCP or SFTP.</p><p>Forgejo Git keys are separate. Signing out of Soda does not sign out of Forgejo or terminate SSH sessions.</p><p><a href="/app/LICENSES.txt">Frontend licenses</a></p></>} />
          <Route path="*" element={<><h1>Page not found</h1><Link to="/">Return to overview</Link></>} />
        </Routes>
      </>}
    </PageSection>
  </Page>;
}

const root = document.getElementById("root");
if (!root) throw new Error("Missing application root");
createRoot(root).render(<StrictMode><BrowserRouter basename="/app"><App /></BrowserRouter></StrictMode>);
