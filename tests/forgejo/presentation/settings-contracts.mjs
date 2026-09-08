// Extract native interaction contracts without freezing presentation markup.
// Template actions are masked while parsing HTML attributes (which can themselves
// contain quoted Go expressions). No runtime routing/configuration consumes this.
export function contracts(source) {
  const actions=[];
  const masked=source.replace(/{{[\s\S]*?}}/g, value => `SODAGO${actions.push(value)-1}END`);
  const restore=value=>value.replace(/SODAGO(\d+)END/g,(_,n)=>actions[+n]);
  const controls=[...masked.matchAll(/<(form|input|textarea|select|button|option)\b([^>]*?)>/g)].map(([,tag,attrs])=>{
    attrs=restore(attrs).replace(/\sclass="(?:{{[\s\S]*?}}|[^"])*"/g,'').replace(/\sautofocus\b/g,'').replace(/\sdata-settings-checklist\b/g,'');
    return `${tag} ${attrs.trim().replace(/\s+/g,' ')}`;
  }).sort();
  const gates=actions.filter(a=>/^{{\s*(?:if |else if |range |with )/.test(a) && !a.includes('ctx.Locale.Tr') && !a.includes('PersonalSettings') && !a.includes('$personal') && !a.includes('PageIsSettingsApplications')).sort();
  return {controls,gates};
}
