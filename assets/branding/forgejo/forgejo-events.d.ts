// Native Forgejo 15 HTMX DOM events used by the notification template hook.
interface DocumentEventMap {
  'htmx:load': CustomEvent<{ elt?: Element }>;
}
interface HTMLElementEventMap {
  'htmx:beforeRequest': CustomEvent<{ xhr: XMLHttpRequest }>;
  'htmx:beforeSwap': CustomEvent<{ xhr: XMLHttpRequest; serverResponse: string }>;
  'htmx:responseError': CustomEvent<{ xhr: XMLHttpRequest }>;
  'htmx:sendError': CustomEvent<{ xhr: XMLHttpRequest }>;
  'htmx:timeout': CustomEvent<{ xhr: XMLHttpRequest }>;
}
interface HTMLAnchorElement {
  _tippy?: { enable(): void; disable(): void; hide(): void };
}
