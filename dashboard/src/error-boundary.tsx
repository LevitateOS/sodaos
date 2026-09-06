import { Component, type ReactNode } from "react";
import { Alert, Button } from "@patternfly/react-core";

export class ErrorBoundary extends Component<{ children: ReactNode }, { failed: boolean }> {
  state = { failed: false };
  static getDerivedStateFromError() { return { failed: true }; }
  render() {
    if (!this.state.failed) return this.props.children;
    // Do not display/log caught exception objects: they may contain provider or
    // user content. Reload clears all in-memory stores and local form drafts.
    return <main><h1>Soda could not display this page</h1><Alert isInline variant="danger" title="Your last operation may have completed. Reload and inspect its actual state before retrying." /><Button onClick={() => window.location.assign("/app/")}>Reload Soda</Button></main>;
  }
}
