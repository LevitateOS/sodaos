import {mountSodaspaces} from './sodaspaces-workspace.js';
import {id} from './sodaspaces-api.js';

export function mountSpacesPage(root: HTMLElement, actor: string) {
  if (!id(actor)) throw Error('Invalid Spaces actor');
  root.classList.add('soda-spaces-page-mount');
  const workspace = mountSodaspaces(root, {kind: 'page', expectedUserId: actor});
  let disposed = false;
  const measure = () => {
    if (disposed || !root.isConnected) return;
    const footer = document.querySelector('footer');
    const top = root.getBoundingClientRect().top + window.scrollY;
    const height = Math.max(0, (window.visualViewport?.height || window.innerHeight) - top - (footer?.getBoundingClientRect().height || 0));
    root.style.height = Math.floor(height) + 'px';
  };
  const observer = new ResizeObserver(measure);
  for (const element of [document.getElementById('navbar'), document.querySelector('footer')]) if (element) observer.observe(element);
  window.addEventListener('resize', measure);
  window.visualViewport?.addEventListener('resize', measure);
  void document.fonts.ready.then(measure);
  measure();
  void workspace.refresh();
  return {get canRestore() { return workspace.canRestore; }, dispose() {
    disposed = true; observer.disconnect();
    window.removeEventListener('resize', measure);
    window.visualViewport?.removeEventListener('resize', measure);
    workspace.dispose();
  }};
}
