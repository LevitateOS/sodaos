import assert from 'node:assert/strict';
import {test} from 'node:test';
import {chromium} from 'playwright';
const origin='http://localhost:3300';
test('login station stays in the desktop left half and avoids compact-pane downloads', {skip:process.env.SODA_FORGEJO_LAYOUT_ORIGIN!==origin},async()=>{
 const browser=await chromium.launch({channel:'chrome',headless:true});
 try {
  for(const theme of ['light','dark']) for(const width of [320,640,960,1199,1200,1440,1920]) {
   const page=await browser.newPage({viewport:{width,height:1000},colorScheme:theme==='light'?'dark':'light'});
   const errors:string[]=[],images:string[]=[];
   page.on('pageerror',e=>errors.push(e.message));
   page.on('response',r=>{if(r.status()>=400)errors.push(r.url());if(r.url().includes('/login-station/'))images.push(r.url());});
   await page.addInitScript(t=>localStorage.setItem('soda.login.theme:/',t),theme);
   const response=await page.goto(`${origin}/user/login`,{waitUntil:'networkidle'});assert.equal(response?.status(),200);
   const layout=await page.evaluate(()=>{
    const story=document.querySelector('.soda-login-story')!,panel=document.querySelector('.soda-login-panel')!;
    const s=story.getBoundingClientRect(),p=panel.getBoundingClientRect();
    return {overflow:document.documentElement.scrollWidth>innerWidth,storyWidth:s.width,panelLeft:p.left,storyRight:s.right,
     image:getComputedStyle(story).backgroundImage,canvas:getComputedStyle(panel).getPropertyValue('--soda-guest-canvas').trim()};
   });
   assert.equal(layout.overflow,false,`${theme}/${width}`);
   assert.equal(images.length,width>=1200?1:0,'photograph downloads only on desktop');
   if(width>=1200){assert(Math.abs(layout.storyWidth-width/2)<1);assert(Math.abs(layout.panelLeft-layout.storyRight)<1);assert(layout.image.includes('approaching-train.webp'));}
   else assert.equal(layout.image,'none');
   for(const name of ['user_name','password']) {
    const input=page.locator(`input[name="${name}"]`);assert(await input.isVisible());
    const box=await input.boundingBox();assert(box&&box.width>=250&&box.x+box.width<=width);
   }
   assert.equal(await page.locator('input[name="password"]').getAttribute('type'),'password');
   assert.equal(await page.locator('form').first().getAttribute('method'),'post');
   assert.deepEqual(errors,[]);
   await page.close();
  }
 }finally{await browser.close();}
});
