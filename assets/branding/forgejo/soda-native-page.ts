import {html, render} from 'lit';
import {connectPage} from '../../../frontend/spaces/soda-connection.js';

const mount = document.getElementById('soda-native-content');
if (mount && mount.dataset.sodaEntryMounted !== 'true') {
  const {documentTitle, actor, view, repositoryId} = mount.dataset;
  if (documentTitle && actor && view) {
    mount.dataset.sodaEntryMounted = 'true';
    document.title = documentTitle;
    let busy = false, mounted = false, generation = 0;
    let owner: {dispose(): void; readonly canRestore: boolean} | undefined;
    window.addEventListener('pagehide', () => {generation++;});
    window.addEventListener('soda-session-retired', () => {
      generation++;
      if (!mounted) {busy = false; failed();}
    });
    window.addEventListener('pageshow', event => {
      if (!event.persisted) return;
      if (!mounted) {generation++; busy = false; failed();}
      else if (owner?.canRestore) {
        // The old controls are irrevocably retired. Dispose their listeners and
        // measurements, then validate the original actor before mounting anew.
        // Runners owns its own restore; uncertain project writes stay retired.
        owner.dispose(); owner = undefined; mounted = false; busy = false;
        void connect(false, true);
      }
    });
    const failed = () => render(html`<p role="alert">Soda connection did not complete. No action was replayed.</p><button class="ui button" type="button" @click=${() => void connect(true)}>Retry connection</button> <a href="./">Back to dashboard</a>`, mount);
    const connect = async (retry = false, restoring = false) => {
      if (busy) return;
      busy = true;
      const current = ++generation;
      const isCurrent = () => generation === current && mount.isConnected;
      render(html`<p role="status">Connecting to Soda…</p>`, mount);
      try {
        if (!await connectPage(actor, view, repositoryId || '', isCurrent, retry, restoring) || !isCurrent()) return;
        if (view === 'spaces') {
          const {mountSpacesPage} = await import('../../../frontend/spaces/sodaspaces-page.js');
          if (generation !== current || !mount.isConnected) return;
          render(html``, mount);
          owner = mountSpacesPage(mount, actor);
        } else if (view === 'runners') {
          const {mountRunnersPage} = await import('../../../frontend/runners/soda-runners-page.js');
          if (generation !== current || !mount.isConnected) return;
          mount.classList.add('soda-settings', 'soda-runner-settings');
          render(html``, mount);
          mountRunnersPage(mount, actor);
        } else if (view === 'repository-spaces') {
          const {mountRepositorySpaces} = await import('../../../frontend/spaces/soda-repository-spaces.js');
          if (generation !== current || !mount.isConnected) return;
          mount.classList.add('soda-settings');
          render(html``, mount);
          owner = mountRepositorySpaces(mount, actor, repositoryId || '');
        }
        mounted = true;
      } catch {
        if (!isCurrent()) return;
        failed();
      } finally {if (generation === current) busy = false;}
    };
    void connect();
  }
}
