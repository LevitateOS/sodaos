import {LitElement, css, html, nothing, render} from 'lit';

class SodaLitSmoke extends LitElement {
  static properties = {
    count: {type: Number},
    label: {type: String},
  };

  static styles = css`
    button { font: inherit; }
  `;

  declare count: number;
  declare label: string;

  constructor() {
    super();
    this.count = 0;
    this.label = '';
  }

  private increment() {
    this.count += 1;
  }

  protected render() {
    return html`
      <button type="button" @click=${this.increment}>${this.label}: ${this.count}</button>
      ${this.count === 0 ? nothing : html`<output>${this.count}</output>`}
    `;
  }
}

if (!customElements.get('soda-lit-smoke')) customElements.define('soda-lit-smoke', SodaLitSmoke);

const standalone = document.querySelector<HTMLElement>('#lit-standalone');
if (!standalone) throw new Error('Missing standalone render target');
render(html`<span>standalone runtime render</span>`, standalone);

await Promise.all(
  [...document.querySelectorAll<SodaLitSmoke>('soda-lit-smoke')].map(element => element.updateComplete),
);
document.body.dataset.litSmokeModules = String(Number(document.body.dataset.litSmokeModules ?? '0') + 1);
