import {html, render} from 'lit';
import {connectPage} from '../../../frontend/spaces/soda-connection.js';

const mount = document.getElementById('soda-native-content');
if (mount) {
  const {documentTitle, actor, view, repositoryId} = mount.dataset;
  if (documentTitle && actor && view) {
    document.title = documentTitle;
    let busy = false, mounted = false, generation = 0;
    window.addEventListener('pagehide', () => {generation++;});
    window.addEventListener('soda-session-retired', () => {
      generation++;
      if (!mounted) {busy = false; failed();}
    });
    window.addEventListener('pageshow', event => {
      if (event.persisted && !mounted) {generation++; busy = false; failed();}
    });
    const failed = () => render(html`<p role="alert">Soda connection did not complete. No action was replayed.</p><button class="ui button" type="button" @click=${() => void connect(true)}>Retry connection</button> <a href="./">Back to dashboard</a>`, mount);
    const connect = async (retry = false) => {
      if (busy) return;
      busy = true;
      const current = ++generation;
      render(html`<p role="status">Connecting to Soda…</p>`, mount);
      try {
        if (!await connectPage(actor, view, repositoryId || '', retry) || generation !== current || !mount.isConnected) return;
        if (view === 'spaces') {
          const {mountSpacesPage} = await import('../../../frontend/spaces/sodaspaces-page.js');
          if (generation !== current) return;
          render(html``, mount);
          mountSpacesPage(mount, actor);
        } else if (view === 'runners') {
          const {mountRunnersPage} = await import('../../../frontend/runners/soda-runners-page.js');
          if (generation !== current) return;
          mount.classList.add('soda-settings', 'soda-runner-settings');
          render(html``, mount);
          mountRunnersPage(mount, actor);
        } else if (view === 'repository-spaces') {
          const {mountRepositorySpaces} = await import('../../../frontend/spaces/soda-repository-spaces.js');
          if (generation !== current) return;
          mount.classList.add('soda-settings');
          render(html``, mount);
          mountRepositorySpaces(mount, actor, repositoryId || '');
        }
        mounted = true;
      } catch {
        if (generation !== current) return;
        failed();
      } finally {if (generation === current) busy = false;}
    };
    void connect();
  }
}
