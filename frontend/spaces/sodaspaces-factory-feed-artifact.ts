import {LitElement, html} from 'lit';
import type {FeedArtifact} from './sodaspaces-factory-feed-response.js';
import type {PreparedExtensionMount} from './soda-extension.js';

export type FeedArtifactRequest = PreparedExtensionMount['request'];
const maxBytes = 8 * 1024 * 1024;
const textBytes = 64 * 1024;

export class SodaFeedArtifact extends LitElement {
  static properties = {artifact: {attribute: false}, request: {attribute: false}, state: {state: true}};
  declare artifact: FeedArtifact;
  declare request: FeedArtifactRequest | undefined;
  declare private state: {busy: boolean; open: boolean; error: string; url: string; text: string; truncated: boolean};
  private cancel?: AbortController;
  constructor() { super(); this.state = {busy: false, open: false, error: '', url: '', text: '', truncated: false}; }
  protected createRenderRoot() { return this; }
  disconnectedCallback() {
    super.disconnectedCallback();
    this.cancel?.abort();
    if (this.state.url) URL.revokeObjectURL(this.state.url);
    this.state = {...this.state, busy: false, open: false, url: '', text: ''};
  }
  private async preview() {
    if (this.state.busy) return;
    if (this.state.url) { this.state = {...this.state, open: !this.state.open}; return; }
    if (!this.request || !this.artifact.href.startsWith('/api/factory/artifacts/')) { this.state = {...this.state, error: 'Open this artifact in its project feed.'}; return; }
    this.state = {...this.state, busy: true, error: ''};
    this.cancel = new AbortController();
    try {
      const response = await this.request(this.artifact.href.slice('/api/'.length), {method: 'GET', signal: this.cancel.signal});
      if (!response.ok) throw Error(response.status === 403 || response.status === 404 ? 'This artifact is no longer available to your account.' : 'The artifact could not be loaded. Try again.');
      if (!response.body || Number(response.headers.get('Content-Length')) > maxBytes) throw Error('This artifact exceeds the preview limit.');
      const reader = response.body.getReader(), chunks: Uint8Array<ArrayBuffer>[] = [];
      let bytes = 0;
      try {
        for (;;) {
          const part = await reader.read();
          if (part.done) break;
          bytes += part.value.byteLength;
          if (bytes > maxBytes) { await reader.cancel(); throw Error('This artifact exceeds the preview limit.'); }
          chunks.push(new Uint8Array(part.value));
        }
      } finally { reader.releaseLock(); }
      const mediaType = this.artifact.mediaType.toLowerCase().split(';')[0] ?? '';
      const blob = new Blob(chunks, {type: mediaType});
      const isText = mediaType.startsWith('text/') || ['application/json', 'application/xml'].includes(mediaType);
      const text = isText ? await blob.slice(0, textBytes).text() : '';
      this.state = {busy: false, open: true, error: '', url: URL.createObjectURL(blob), text, truncated: isText && bytes > textBytes};
    } catch (error) {
      if (this.cancel.signal.aborted) return;
      this.state = {...this.state, busy: false, error: error instanceof Error ? error.message : 'The artifact could not be loaded.'};
    }
  }
  protected render() {
    const artifact = this.artifact;
    if (!artifact) return html``;
    const type = artifact.mediaType.toLowerCase();
    return html`<section class="soda-feed-artifact" aria-label=${artifact.name}>
      <header><div><strong>${artifact.name}</strong><span>${type}</span></div><button type="button" @click=${() => void this.preview()} ?disabled=${this.state.busy} aria-expanded=${this.state.open}>${this.state.busy ? 'Loading…' : this.state.open ? 'Hide preview' : 'Preview'}</button></header>
      ${this.state.error ? html`<p role="alert">${this.state.error}</p>` : ''}
      ${this.state.open && this.state.url ? html`<div class="soda-feed-artifact-preview">
        ${/^image\/(png|jpeg|webp|gif|avif)$/.test(type) ? html`<img src=${this.state.url} alt=${artifact.name}>`
          : /^video\/(mp4|webm|ogg)$/.test(type) ? html`<video controls preload="metadata" src=${this.state.url} aria-label=${artifact.name}></video>`
          : /^audio\/(mpeg|mp4|webm|ogg|wav|x-wav)$/.test(type) ? html`<audio controls preload="metadata" src=${this.state.url} aria-label=${artifact.name}></audio>`
          : type.startsWith('text/') || ['application/json', 'application/xml'].includes(type) ? html`<pre tabindex="0">${this.state.text || '(Empty file)'}</pre>${this.state.truncated ? html`<p class="soda-feed-block-note">Showing the first 64 KiB. Download for the complete file.</p>` : ''}`
          : html`<p>This file is available to download.</p>`}
        <a href=${this.state.url} download=${artifact.name}>Download ${artifact.name}</a>
      </div>` : ''}
    </section>`;
  }
}
if (typeof customElements !== 'undefined' && !customElements.get('soda-feed-artifact')) customElements.define('soda-feed-artifact', SodaFeedArtifact);
declare global { interface HTMLElementTagNameMap { 'soda-feed-artifact': SodaFeedArtifact; } }
