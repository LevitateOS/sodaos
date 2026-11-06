// Native-header Fountain composition over the same production issue components
// used by the standalone application frame.
import {html, render} from 'lit';
import {ifDefined} from 'lit/directives/if-defined.js';
import {SodaFactoryIssues} from '@factory/product/frontend/factory/issues/issues.js';
import {SodaFactoryPulls} from '@factory/product/frontend/factory/pulls/list.js';
import {SodaFactoryPull} from '@factory/product/frontend/factory/pulls/detail.js';
import {check, id} from '@factory/product/frontend/factory/api/models.js';
import {markdownLink} from '@factory/product/frontend/factory/shared/markdown.js';
import {SodaFactoryIssueCreate} from '@factory/product/frontend/factory/issues/create.js';
import {mountIssueScreen} from '@factory/product/frontend/factory/issues/screen.js';
import {fountainIssueContexts} from './soda-factory-issue-host.js';
import {fountainAcceptanceLinks, fountainWorkLinks, nativeIssueLink, nativePullLink} from './soda-factory-links.js';
import {fountainCommandScope} from './soda-factory-feed-host.js';
import {withFactoryRepository} from './soda-place-view.js';
import type {WorkPlace} from './soda-places.js';
import '@factory/product/frontend/factory/work/acceptance.js';
import '@factory/product/frontend/factory/work/context-resolution.js';
import '@factory/product/frontend/factory/work/completion-retry.js';

export const mountIssuesPlace = withFactoryRepository((host, context, initial) => {
  if (initial.place !== 'issues' && initial.place !== 'pulls') throw Error('Invalid issue destination');
  let place: WorkPlace = initial;
  let active = true;
  let disposed = false;
  let lifetime = new AbortController();
  let release: (() => void) | undefined;
  let controls: HTMLElement | undefined;
  let available = false;
  let content = document.createElement('div');
  let ports: Awaited<ReturnType<typeof fountainIssueContexts>> | undefined;
  // Native link helpers validate decoded paths; the place router owns encoding.
  const repositoryPath = `${context.prefix}/${context.slug}`;
  const scroll = () => {
    if (!active || disposed) return;
    const target = place.fragment?.slice(1) || (place.tab === 'factory' ? 'factory' : '');
    if (!target) return;
    let fragment: string;
    try {
      fragment = decodeURIComponent(target);
    } catch {
      return;
    }
    [...host.querySelectorAll<HTMLElement>('[id]')].find((element) => element.id === fragment)?.scrollIntoView();
  };
  const drawControls = () => {
    if (!controls) return;
    const issue = place.item;
    const live =
      available && active && ports && issue && issue !== 'new'
        ? {transport: ports.list.transport, repository: context.factoryRepository, issue}
        : undefined;
    const commandScope = (signal: AbortSignal) => fountainCommandScope(ports!.list.transport, signal);
    render(
      live
        ? html`<details class="soda-issue-factory-controls" ?open=${place.tab === 'factory'} id="factory">
            <summary>Factory controls</summary>
            <soda-factory-requirements-acceptance
              .context=${{...live, links: fountainAcceptanceLinks(repositoryPath), commandScope}}
            ></soda-factory-requirements-acceptance>
            <soda-factory-context-resolution .context=${{...live, commandScope}}></soda-factory-context-resolution>
            <soda-factory-completion-retry .context=${{...live, commandScope}}></soda-factory-completion-retry>
          </details>`
        : html``,
      controls
    );
  };
  const stop = () => {
    lifetime.abort();
    available = false;
    release?.();
    release = undefined;
    if (controls) render(html``, controls);
    controls = undefined;
    ports = undefined;
    render(html``, content);
    content.remove();
  };
  const load = () => {
    stop();
    if (!active || disposed) return;
    content = document.createElement('div');
    host.append(content);
    lifetime = new AbortController();
    const signal = lifetime.signal;
    const current = () => !disposed && active && lifetime.signal === signal && !signal.aborted;
    const native =
      place.place === 'pulls'
        ? place.item && place.item !== 'new'
          ? nativePullLink(repositoryPath, place.item)
          : `${repositoryPath}/${place.item === 'new' ? 'compare' : 'pulls'}`
        : `${repositoryPath}/issues/new`;
    render(html`<p role="status">Loading repository information…</p>`, content);
    void fountainIssueContexts(
      context.transport,
      {
        host: context.factoryHost,
        actor: context.factoryActor,
        repository: context.factoryRepository,
        nativeRepository: context.repository,
        name: context.slug,
        prefix: context.prefix,
        origin: location.origin,
        issuePath: (number) => context.places.issues(number),
        navigate(path, opening) {
          opening.throwIfAborted();
          context.navigate(path);
        },
        openExternal(url, opening) {
          opening.throwIfAborted();
          window.open(url, '_blank', 'noopener,noreferrer');
        },
      },
      signal
    )
      .then(
        (admitted) => {
          if (!current()) return;
          ports = admitted;
          render(html``, content);
          if (place.place === 'pulls') {
            const navigation = document.createElement('nav');
            navigation.setAttribute('aria-label', 'Pull request navigation');
            render(html`<a href=${context.places.pulls()}>Pull requests</a>
              ${place.item && place.item !== 'new' ? html` <a href=${ifDefined(nativePullLink(repositoryPath, place.item))}>Open in Fountain</a>` : ''}`, navigation);
            content.append(navigation);
            if (!place.item || place.item === 'new') {
              if (place.item === 'new') {
                const notice = document.createElement('section');
                render(html`<p>Choose a branch pair in Fountain to create a pull request.</p><a href=${ifDefined(native)}>Open compare view</a>`, notice);
                content.append(notice);
                return;
              }
              const list = new SodaFactoryPulls();
              list.context = {...admitted.list, openPull: (pull, opening) => {
                opening.throwIfAborted();
                check(current() && pull.kind === 'pull' && pull.repository === context.factoryRepository && id(pull.number));
                context.navigate(context.places.pulls(pull.number));
              }};
              release = () => { list.context = undefined; list.remove(); };
              content.append(list);
              return;
            }
            const view = new SodaFactoryPull();
            const page = new URL(context.places.pulls(place.item), location.origin);
            view.context = {
              transport: admitted.list.transport, host: context.factoryHost, actor: context.factoryActor,
              repository: context.factoryRepository, pull: place.item, scope: admitted.list.scope,
              initialTab: place.tab === 'commits' || place.tab === 'reviews' || place.tab === 'threads' || place.tab === 'files' || place.tab === 'checks' ? place.tab : 'conversation',
              navigateTab(tab, opening) {
                opening.throwIfAborted(); check(current());
                context.navigate(context.places.pulls(place.item!, tab));
              },
              commandScope: (signal) => fountainCommandScope(admitted.list.transport, signal),
              resolveLink(href) {
                if (!href || /[\\\u0000-\u001f\u007f]/.test(href)) return;
                return markdownLink(new URL(href, href.startsWith('#') || href.startsWith('?') ? page : new URL(repositoryPath + '/', location.origin)).href);
              },
              openLink(url, opening) {
                opening.throwIfAborted(); check(current());
                const target = new URL(url);
                if (target.origin === location.origin) context.navigate(target.pathname + target.search + target.hash);
                else window.open(target.href, '_blank', 'noopener,noreferrer');
              },
            };
            release = () => { view.context = undefined; view.remove(); };
            content.append(view);
            return;
          }
          const navigation = document.createElement('nav');
          navigation.setAttribute('aria-label', 'Issue navigation');
          render(
            html`<a href=${context.places.issues()}>Issues</a>${
                place.item && place.item !== 'new'
                  ? html` <a href=${context.places.issues(place.item)}>Conversation</a>
                      <a href=${context.places.factoryStatus(place.item)}>Factory</a>
                      <a href=${ifDefined(nativeIssueLink(repositoryPath, place.item))}>Open in Fountain</a>`
                  : html` <a href=${context.places.issues('new')}>New issue</a>`
              }`,
            navigation
          );
          content.append(navigation);
          if (place.item === 'new') {
            const form = new SodaFactoryIssueCreate();
            form.context = {
              transport: admitted.list.transport,
              host: context.factoryHost,
              actor: context.factoryActor,
              repository: context.factoryRepository,
              scope: admitted.list.scope,
              commandScope: (signal) => fountainCommandScope(admitted.list.transport, signal),
              openIssue: (issue, opening) => admitted.list.openIssue(issue, opening),
            };
            release = () => {
              form.context = undefined;
              form.remove();
            };
            content.append(form);
            return;
          }
          if (!place.item) {
            const list = new SodaFactoryIssues();
            list.context = admitted.list;
            release = () => {
              list.context = undefined;
              list.remove();
            };
            content.append(list);
            return;
          }
          controls = document.createElement('div');
          const screen = mountIssueScreen(
            admitted.detail(place.item),
            {
              transport: admitted.list.transport,
              repository: context.factoryRepository,
              issue: place.item,
              links: fountainWorkLinks(repositoryPath),
            },
            signal,
            current,
            (value) => {
              available = value;
              drawControls();
              if (value)
                requestAnimationFrame(() => {
                  if (current()) scroll();
                });
            },
            {
              transport: admitted.list.transport,
              host: context.factoryHost,
              actor: context.factoryActor,
              repository: context.factoryRepository,
              issue: place.item,
              scope: admitted.list.scope,
              commandScope: (signal) => fountainCommandScope(admitted.list.transport, signal),
            },
            {
              transport: admitted.list.transport,
              host: context.factoryHost,
              actor: context.factoryActor,
              repository: context.factoryRepository,
              issue: place.item,
              scope: admitted.list.scope,
              commandScope: (signal) => fountainCommandScope(admitted.list.transport, signal),
            },
            {
              transport: admitted.list.transport,
              host: context.factoryHost,
              actor: context.factoryActor,
              repository: context.factoryRepository,
              issue: place.item,
              scope: admitted.list.scope,
              commandScope: (signal) => fountainCommandScope(admitted.list.transport, signal),
            }
          );
          release = screen.dispose;
          content.append(screen.element, controls);
        },
        () => {
          if (current())
            render(html`<p role="status">Repository information is unavailable. Refresh to try again.</p>`, content);
        }
      )
      .catch(() => {
        if (current()) {
          stop();
          content = document.createElement('div');
          host.append(content);
          render(html`<p role="status">This issue view could not load.</p>`, content);
        }
      });
  };
  load();
  return {
    update(next) {
      if (next.place !== 'issues' && next.place !== 'pulls') return;
      const same = next.place === place.place && next.item === place.item;
      const changedPullTab = next.place === 'pulls' && next.tab !== place.tab;
      place = next;
      if (same && !changedPullTab) {
        const panel = controls?.querySelector('details');
        if (panel) panel.open = place.tab === 'factory';
        scroll();
      } else load();
    },
    setActive(value) {
      if (active === value) return;
      active = value;
      if (value) load();
      else stop();
    },
    dispose() {
      disposed = true;
      stop();
    },
  };
});
