import {html, render} from 'lit';

// The native template supplies a validated presentation, never Soda authority.
// Existing protected pages remain the destination until their controls move here.
const mount = document.getElementById('soda-native-content');
if (mount) {
  const {title, documentTitle, destination} = mount.dataset;
  if (title && documentTitle && destination) {
    document.title = documentTitle;
    render(html`<p><a class="ui primary button" href=${destination}>Open ${title}</a></p>`, mount);
  }
}
