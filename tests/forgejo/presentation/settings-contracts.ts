import assert from 'node:assert/strict';
// Extract native interaction contracts without freezing presentation markup.
// Template actions are masked while parsing HTML attributes (which can themselves
// contain quoted Go expressions). No runtime routing/configuration consumes this.
export function contracts(source: string) {
  // Tag creation shares the inventory landing page. Only its dedicated edit
  // state keeps autofocus; this reviewed focus delta is not a form-field change.
  source=source.replace(/{{if \.PageIsEditProtectedTag}}autofocus\s*{{end}}/g,'');
  const actions: string[]=[];
  const masked=source.replace(/{{[\s\S]*?}}/g, value => `SODAGO${actions.push(value)-1}END`);
  const restore=(value: string)=>value.replace(/SODAGO(\d+)END/g,(_,n: string)=>{ const action=actions[+n]; assert(action !== undefined); return action; });
  const controls=[...masked.matchAll(/<(form|input|textarea|select|button|option)\b([^>]*?)>/g)].map(([,tag,attrs])=>{
    assert(tag && attrs !== undefined);
    attrs=restore(attrs).replace(/\sclass="(?:{{[\s\S]*?}}|[^"])*"/g,'').replace(/\sautofocus\b/g,'').replace(/\sdata-settings-checklist\b/g,'');
    return `${tag} ${attrs.trim().replace(/\s+/g,' ')}`;
  }).sort();
  const gates=actions.filter(a=>/^{{\s*(?:if |else if |range |with )/.test(a) && !a.includes('ctx.Locale.Tr') && !a.includes('PersonalSettings') && !a.includes('SettingsPresentation') && !a.includes('$settings') && !a.includes('$personal') && !a.includes('PageIsSettingsApplications')).sort();
  return {controls,gates};
}
