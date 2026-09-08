import assert from 'node:assert/strict';
import {test} from 'node:test';
import {readFile} from 'node:fs/promises';
import {createHash} from 'node:crypto';
const root=new URL('../../appliance/forgejo/templates/',import.meta.url);
test('milestone redesign changes only list items in both callers',async()=>{
 const expected=JSON.parse(await readFile(new URL('milestone-page-boundaries.json',import.meta.url)));
 for(const [path,hash] of Object.entries(expected)) {
  const source=await readFile(new URL(path,root),'utf8');
  const outside=source.replace(/{{template "custom\/soda\/milestone_row"[^\n]*}}/,'MILESTONE_ROW');
  assert.equal(createHash('sha256').update(outside).digest('hex'),hash,path+': heading, filters, navigation or other page content changed');
 }
 const row=await readFile(new URL('custom/soda/milestone_row.tmpl',root),'utf8');
 for(const marker of ['.Completeness','.NumOpenIssues','.NumClosedIssues','.TotalTrackedTime','.UpdatedUnix','.IsClosed','.ClosedDateUnix','.DeadlineString','.IsOverdue','.RenderedContent','{{if $editable}}','{{$page.Link}}/{{.ID}}/edit','{{$page.Link}}/{{.ID}}/open','{{$page.Link}}/{{.ID}}/close','{{$page.RepoLink}}/milestones/delete','data-modal-id="delete-milestone"','aria-label="{{.Name}}"']) assert(row.includes(marker),marker);
 const repo=await readFile(new URL('repo/issue/milestones.tmpl',root),'utf8');
 assert(repo.includes('"Editable" (and (or $.CanWriteIssues $.CanWritePulls) (not $.Repository.IsArchived))'));
 const global=await readFile(new URL('user/dashboard/milestones.tmpl',root),'utf8');
 assert(global.includes('"Editable" false'));
});
