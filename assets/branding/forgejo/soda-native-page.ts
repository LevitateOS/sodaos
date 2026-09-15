import {html, render} from 'lit';
import {connectPage} from '../../../frontend/spaces/soda-connection.js';

type PageOwner = {dispose(): void; readonly canRestore: boolean};
type ConnectedSession = Exclude<Awaited<ReturnType<typeof connectPage>>, false>;

type NativePage = {
  mount: HTMLElement;
  actor: string;
  view: string;
  repositoryId: string;
  busy: boolean;
  mounted: boolean;
  generation: number;
  owner: PageOwner | undefined;
};

function generationIsCurrent(page: NativePage, current: number): boolean {
  return page.generation === current && page.mount.isConnected;
}

function renderFailed(page: NativePage): void {
  render(
    html`<p role="alert">Soda connection did not complete. No action was replayed.</p>
      <button class="ui button" type="button" @click=${() => void connectNativePage(page, true)}>
        Retry connection
      </button>
      <a href="./">Back to dashboard</a>`,
    page.mount
  );
}

async function loadSpacesView(
  page: NativePage,
  session: ConnectedSession,
  isCurrent: () => boolean
): Promise<PageOwner | undefined> {
  const {mountSpacesPage} = await import('../../../frontend/spaces/sodaspaces-page.js');
  if (!isCurrent()) return undefined;
  render(html``, page.mount);
  return mountSpacesPage(page.mount, page.actor, session);
}

async function loadRunnersView(page: NativePage, session: ConnectedSession, isCurrent: () => boolean): Promise<void> {
  const {mountRunnersPage} = await import('../../../frontend/runners/soda-runners-page.js');
  if (!isCurrent()) return;
  page.mount.classList.add('soda-settings', 'soda-runner-settings');
  render(html``, page.mount);
  mountRunnersPage(page.mount, page.actor, session);
}

async function loadTailnetView(page: NativePage, session: ConnectedSession, isCurrent: () => boolean): Promise<void> {
  const {mountTailnetPage} = await import('../../../frontend/tailnet/soda-tailnet-page.js');
  if (!isCurrent()) return;
  page.mount.classList.add('soda-settings', 'soda-tailnet-settings');
  render(html``, page.mount);
  mountTailnetPage(page.mount, page.actor, session);
}

async function loadRepositorySpacesView(
  page: NativePage,
  session: ConnectedSession,
  isCurrent: () => boolean
): Promise<PageOwner | undefined> {
  const {mountRepositorySpaces} = await import('../../../frontend/spaces/soda-repository-spaces.js');
  if (!isCurrent()) return undefined;
  page.mount.classList.add('soda-settings');
  render(html``, page.mount);
  return mountRepositorySpaces(page.mount, page.actor, page.repositoryId, session);
}

async function loadNativeView(
  page: NativePage,
  session: ConnectedSession,
  isCurrent: () => boolean
): Promise<PageOwner | undefined> {
  if (page.view === 'spaces') return loadSpacesView(page, session, isCurrent);
  if (page.view === 'runners') {
    await loadRunnersView(page, session, isCurrent);
    return undefined;
  }
  if (page.view === 'tailnet') {
    await loadTailnetView(page, session, isCurrent);
    return undefined;
  }
  if (page.view === 'repository-spaces') return loadRepositorySpacesView(page, session, isCurrent);
  return undefined;
}

async function mountConnectedPage(
  page: NativePage,
  retry: boolean,
  restoring: boolean,
  isCurrent: () => boolean
): Promise<void> {
  const session = await connectPage(page.actor, page.view, page.repositoryId, isCurrent, retry, restoring);
  if (!session || !isCurrent()) return;
  const owner = await loadNativeView(page, session, isCurrent);
  if (!isCurrent()) return;
  if (owner !== undefined) page.owner = owner;
  page.mounted = true;
}

function isCurrentGeneration(page: NativePage, current: number): () => boolean {
  return () => generationIsCurrent(page, current);
}

async function connectNativePage(page: NativePage, retry = false, restoring = false): Promise<void> {
  if (page.busy) return;
  page.busy = true;
  const current = ++page.generation;
  const isCurrent = isCurrentGeneration(page, current);
  render(html`<p role="status">Connecting to Soda…</p>`, page.mount);
  try {
    await mountConnectedPage(page, retry, restoring, isCurrent);
  } catch {
    if (isCurrent()) renderFailed(page);
  } finally {
    if (page.generation === current) page.busy = false;
  }
}

function restoreNativePage(page: NativePage, event: PageTransitionEvent): void {
  if (!event.persisted) return;
  if (!page.mounted) {
    page.generation++;
    page.busy = false;
    renderFailed(page);
    return;
  }
  if (!page.owner?.canRestore) return;
  // The old controls are irrevocably retired. Dispose their listeners and
  // measurements, then validate the original actor before mounting anew.
  // Runners owns its own restore; in-flight project writes stay retired.
  page.owner.dispose();
  page.owner = undefined;
  page.mounted = false;
  page.busy = false;
  void connectNativePage(page, false, true);
}

function bindNativePageLifetime(page: NativePage): void {
  window.addEventListener('pagehide', () => {
    page.generation++;
  });
  window.addEventListener('soda-session-retired', () => {
    page.generation++;
    if (!page.mounted) {
      page.busy = false;
      renderFailed(page);
    }
  });
  window.addEventListener('pageshow', (event) => {
    restoreNativePage(page, event);
  });
}

function startNativePage(): void {
  const mount = document.getElementById('soda-native-content');
  if (!mount || mount.dataset.sodaEntryMounted === 'true') return;
  const {documentTitle, actor, view, repositoryId} = mount.dataset;
  if (!documentTitle || !actor || !view) return;
  mount.dataset.sodaEntryMounted = 'true';
  document.title = documentTitle;
  const page: NativePage = {
    mount,
    actor,
    view,
    repositoryId: repositoryId || '',
    busy: false,
    mounted: false,
    generation: 0,
    owner: undefined,
  };
  bindNativePageLifetime(page);
  void connectNativePage(page);
}

startNativePage();
