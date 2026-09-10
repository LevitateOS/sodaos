import {html, render} from 'lit';
import {connectPage} from '../../../frontend/spaces/soda-connection.js';

const mount = document.getElementById('soda-native-content');
if (mount) {
  const {title, documentTitle, destination, actor, view, repositoryId} = mount.dataset;
  if (title && documentTitle && destination && actor && view) {
    document.title = documentTitle;
    let busy = false;
    const connect = async (retry = false) => {
      if (busy) return;
      busy = true;
      render(html`<p role="status">Connecting to Soda…</p>`, mount);
      try {
        if (await connectPage(actor, view, repositoryId || '', retry))
          render(html`<p><a class="ui primary button" href=${destination}>Open ${title}</a></p>`, mount);
      } catch {
        render(html`<p role="alert">Soda connection did not complete. No action was replayed.</p><button class="ui button" type="button" @click=${() => void connect(true)}>Retry connection</button> <a href="./">Back to dashboard</a>`, mount);
      } finally {busy = false;}
    };
    void connect();
  }
}
