import { StrictMode, useEffect } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, Link, Route, Routes, useLocation } from "react-router-dom";
import { Alert, Button, Page, PageSection, Spinner } from "@patternfly/react-core";
import "@patternfly/react-core/dist/styles/base.css";
import "./style.css";
import logo from "../../assets/branding/source/soda-logo-horizontal.svg";
import { useSession } from "./session";
import { Profile } from "./profile";
import { RepositoryList, CreateRepository, RepositoryDetail } from "./repositories";
import { EnvironmentList, EnvironmentDetail } from "./environments";
import { ForgejoAccount, People } from "./accounts";
import { ErrorBoundary } from "./error-boundary";
import { History, CommitDetail, Refs, Compare } from "./history";
import { FileEditor } from "./file-editor";
import { ForkRepository, ImportRepository } from "./repository-copy";
import { Issues, NewIssue, IssueDetail } from "./issues";
import { Labels, Milestones } from "./issue-metadata";

export function App() {
  const { phase, session, error, load, logout } = useSession();
  const location = useLocation();
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
          <Link to="/repositories">Repositories</Link><Link to="/environments">Environments</Link><Link to="/account">Forgejo account</Link>
          <a href={session.forgejo_url}>Open Forgejo</a>
          <Button variant="link" onClick={() => void logout()}>Sign out of Soda</Button>
        </nav>
        <Routes key={`${session.user.id}:${location.pathname}`}>
          <Route path="/" element={<><h1>Welcome, {session.user.soda_display_name || session.user.login}</h1><p>Discover or create a Forgejo repository, create its persistent environment, then explicitly join using your public development-access key.</p><p><Link to="/repositories">Browse repositories</Link> · <Link to="/environments">Discover environments</Link></p><Alert isInline variant="warning" title="This preview has not been built or installed-verified. Direct project routing and native workload/persistence proof remain pending." /></>} />
          <Route path="/repositories" element={<RepositoryList session={session} />} />
          <Route path="/repositories/new" element={<CreateRepository session={session} />} />
          <Route path="/repositories/import" element={<ImportRepository session={session} />} />
          <Route path="/repositories/:owner/:repo/history" element={<History session={session} />} />
          <Route path="/repositories/:owner/:repo/commits/:sha" element={<CommitDetail session={session} />} />
          <Route path="/repositories/:owner/:repo/refs" element={<Refs session={session} />} />
          <Route path="/repositories/:owner/:repo/compare" element={<Compare session={session} />} />
          <Route path="/repositories/:owner/:repo/edit" element={<FileEditor session={session} />} />
          <Route path="/repositories/:owner/:repo/fork" element={<ForkRepository session={session} />} />
          <Route path="/repositories/:owner/:repo/issues" element={<Issues session={session} />} />
          <Route path="/repositories/:owner/:repo/issues/new" element={<NewIssue session={session} />} />
          <Route path="/repositories/:owner/:repo/issues/:index" element={<IssueDetail session={session} />} />
          <Route path="/repositories/:owner/:repo/labels" element={<Labels session={session} />} />
          <Route path="/repositories/:owner/:repo/milestones" element={<Milestones session={session} />} />
          <Route path="/repositories/:owner/:repo" element={<RepositoryDetail session={session} />} />
          <Route path="/environments" element={<EnvironmentList session={session} />} />
          <Route path="/environments/:id" element={<EnvironmentDetail session={session} />} />
          <Route path="/account" element={<ForgejoAccount session={session} />} />
          <Route path="/administration/people" element={<People session={session} />} />
          <Route path="/profile" element={<Profile key={session.user.id} session={session} />} />
          <Route path="/help" element={<><h1>Development access</h1><p>Register only public SSH keys. Adding a key does not rotate keys already installed in environments. Explicitly join through Environments, then use the displayed project IP with SSH, SCP or SFTP.</p><p>Forgejo Git keys are separate. Signing out of Soda does not sign out of Forgejo or terminate SSH sessions.</p><p><a href="/app/LICENSES.txt">Frontend licenses</a></p></>} />
          <Route path="*" element={<><h1>Page not found</h1><Link to="/">Return to overview</Link></>} />
        </Routes>
      </>}
    </PageSection>
  </Page>;
}

const root = document.getElementById("root");
if (!root) throw new Error("Missing application root");
createRoot(root).render(<StrictMode><ErrorBoundary><BrowserRouter basename="/app"><App /></BrowserRouter></ErrorBoundary></StrictMode>);
