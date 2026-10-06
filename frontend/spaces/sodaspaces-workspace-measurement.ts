import type {LitElement, ReactiveController} from 'lit';

// Only subscriptions live here. Geometry, layout and minimum-size publication
// remain with SodaSpaces. A disconnected workspace is permanently disposed, so
// retirement must also block late font promises and already queued observer calls.
export class WorkspaceMeasurement implements ReactiveController {
  private observer: ResizeObserver | undefined;
  private lifetime: AbortController | undefined;
  private retired = false;

  constructor(
    private readonly host: LitElement,
    private readonly measure: () => void
  ) {
    host.addController(this);
  }

  hostUpdated() {
    if (this.retired || this.observer || !this.host.isConnected) return;
    // Connection precedes the first render. Wait for the actual canvas, and
    // retry on a later update if the initial render did not contain it.
    const canvas = this.host.querySelector('.soda-workspace-canvas');
    if (!canvas) return;
    const lifetime = (this.lifetime = new AbortController());
    const notify = () => {
      if (!lifetime.signal.aborted && this.host.isConnected) this.measure();
    };
    this.observer = new ResizeObserver(notify);
    this.observer.observe(canvas);
    this.observer.observe(this.host);
    document.fonts.addEventListener('loadingdone', notify, {signal: lifetime.signal});
    window.visualViewport?.addEventListener('resize', notify, {signal: lifetime.signal});
    void document.fonts.ready.then(notify);
  }

  hostDisconnected() {
    this.retire();
  }

  retire() {
    this.retired = true;
    this.observer?.disconnect();
    this.observer = undefined;
    this.lifetime?.abort();
  }
}
