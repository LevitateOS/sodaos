import assert from 'node:assert/strict';
import {test} from 'node:test';
import {chromium} from 'playwright';
const origin='http://localhost:8140';
const review=process.env.SODA_FORGEJO_REVIEW_ORIGIN===origin;
const enabled=review||process.env.SODA_FORGEJO_LAYOUT_ORIGIN==='http://localhost:3300';
const fixture=(name:string,theme:string)=>review?`${origin}/${name}-${theme}.html`:new URL(`../../../.artifacts/forgejo-presentation/${name}-${theme}.html`,import.meta.url).href;
const luminance=(rgb:string)=>{const c=rgb.match(/[\d.]+/g)!.slice(0,3).map(v=>Number(v)/255).map(v=>v<=.04045?v/12.92:((v+.055)/1.055)**2.4);return c[0]!*.2126+c[1]!*.7152+c[2]!*.0722;};
const contrast=(a:string,b:string)=>{const x=luminance(a),y=luminance(b);return (Math.max(x,y)+.05)/(Math.min(x,y)+.05);};
test('refined page families fit narrow screens and keep content readable',{skip:!enabled},async()=>{
 const browser=await chromium.launch({channel:'chrome',headless:true});
 try {
  const page=await browser.newPage();
  const errors:string[]=[];page.on('pageerror',e=>errors.push(e.message));
  page.on('response',r=>{if(r.status()>=400)errors.push(`${r.status()} ${r.url()}`);});
  for(const theme of ['light','dark']) for(const width of [320,390,640,720,768,800,960,1100,1101,1440]) {
   await page.setViewportSize({width,height:1000});
   for(const name of (review?['migration','refinement','signin','home']:['migration','refinement','signin'])) {
    const response=await page.goto(fixture(name,theme));if(review)assert.equal(response?.status(),200);
    await page.evaluate(()=>document.fonts.ready);
    assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true,`${name}/${theme}/${width}: page overflow`);
    assert.equal(await page.locator('[id=""]').count(),0,'no empty IDs from optional intro context');
    if(name==='home') {
     const action=await page.locator('.soda-home-primary').evaluate(el=>({border:getComputedStyle(el).borderTopWidth,transform:getComputedStyle(el).textTransform}));
     assert.deepEqual(action,{border:'2px',transform:'uppercase'},'standalone homepage uses shared action styling');
    }
    if(name==='migration') {
     const cards=page.locator('.soda-migrate-provider');assert.equal(await cards.count(),4);
     for(const card of await cards.all()) {
      const rect=await card.boundingBox();assert(rect&&rect.x>=0&&rect.x+rect.width<=width);
     }
     const git=page.locator('.soda-migrate-provider--git');await git.hover();
     await page.waitForTimeout(150);
     const colors=await git.evaluate(el=>({bg:getComputedStyle(el).backgroundColor,fg:getComputedStyle(el.querySelector('p')!).color}));
     assert(contrast(colors.bg,colors.fg)>=4.5,`Git source description must stay readable on hover: ${theme}/${width} ${JSON.stringify(colors)}`);
     await git.focus();assert.equal(await git.evaluate(el=>getComputedStyle(el).outlineStyle),'solid');
     if(width<=768) assert.equal(await page.locator('.soda-migrate-story').evaluate(el=>getComputedStyle(el).gridTemplateColumns),'none','no vacant artwork column');
    }
    if(name==='refinement') {
     const button=page.locator('#long-label');const box=await button.boundingBox();assert(box);
     assert(box.height>=44);if(width<=390)assert(box.height>44,'long label should grow instead of clipping');
     assert(await button.evaluate(el=>el.scrollHeight<=el.clientHeight));
     assert.equal(await page.locator('#admin-reference .vertical.menu a').last().evaluate(el=>getComputedStyle(el).whiteSpace),'normal');
     const formColumns=await page.locator('#creation-reference').evaluate(el=>getComputedStyle(el).gridTemplateColumns.split(' ').length);
     assert.equal(formColumns,width<1101?1:2);
     const hr=page.locator('#long-controls hr');assert.equal(await hr.isVisible(),false,'action row owns the single separator');
     await page.locator('.soda-quota-overview summary').click();assert(await page.locator('.soda-quota-overview ul').isVisible());
     assert.equal(await page.locator('button[disabled]').isDisabled(),true);
     // Actual native empty-feed SVG adapter: its icon must have drawable space.
     await page.locator('#long-controls').evaluate(el=>el.insertAdjacentHTML('beforeend','<div class="soda-list"><div id="empty-feed"><svg width="48" height="48" viewBox="0 0 16 16"><path d="M0 0h16v16H0Z"/></svg></div></div>'));
     const icon=await page.locator('#empty-feed > svg').evaluate(el=>({width:el.getBoundingClientRect().width,padding:getComputedStyle(el).padding}));assert.deepEqual(icon,{width:32,padding:'0px'});
    }
    if(name==='signin'&&width<900) {
     const intro=await page.locator('.soda-login-eyebrow').boundingBox();assert(intro&&intro.y<160,'mobile form should follow the identity directly');
    }
   }
  }
  // Avatar styles with shared role names must not cross page boundaries.
  await page.goto(fixture('refinement','light'));
  await page.setViewportSize({width:1440,height:1000});
  await page.locator('main').evaluate(el=>el.insertAdjacentHTML('beforeend','<div class="soda-account-details--profile"><div class="soda-profile-portrait"><img id="editor-portrait" alt="Fixture avatar" src="http://localhost:3300/assets/soda/source/soda-symbol-brutalist.svg"></div></div><div class="soda-profile-card-context"><div class="soda-profile-portrait"><img id="public-portrait" alt="Fixture avatar" src="http://localhost:3300/assets/soda/source/soda-symbol-brutalist.svg"></div></div>'));
  await page.locator('#editor-portrait').evaluate(el => (el as HTMLImageElement).decode());
  assert.equal(await page.locator('#editor-portrait').evaluate(el=>el.getBoundingClientRect().width),112);
  assert.equal(await page.locator('#public-portrait').evaluate(el=>el.getBoundingClientRect().width),72);
  await page.setViewportSize({width:390,height:1000});
  assert.equal(await page.locator('#editor-portrait').evaluate(el=>el.getBoundingClientRect().width),64);
  assert.equal(await page.locator('#public-portrait').evaluate(el=>el.getBoundingClientRect().width),56);
  assert.deepEqual(errors,[]);
 }finally{await browser.close();}
});
