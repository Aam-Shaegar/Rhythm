import test from 'node:test';
import assert from 'node:assert/strict';
import {compareTasks,taskStatus} from '../src/tasks.ts';
import {deriveNotices} from '../src/notifications.ts';
test('unfinished first, then completed, with deadlines ascending in each group',()=>{
 const items=[{id:'late-done',is_completed:true,due_at:'2026-11-10T09:00:00Z'},{id:'later',is_completed:false,due_at:'2026-11-11T09:00:00Z'},{id:'earlier',is_completed:false,due_at:'2026-11-10T12:00:00Z'}];
 assert.deepEqual(items.sort(compareTasks).map(t=>t.id),['earlier','later','late-done']);
});
test('late completion stays overdue after completion; same-day missed deadline is overdue',()=>{
 assert.match(taskStatus({is_completed:true,due_at:'2026-11-10T09:00:00Z',completed_at:'2026-11-10T09:01:00Z'}),/опозданием/);
 assert.equal(taskStatus({is_completed:false,due_at:'2026-11-10T09:00:00Z'},Date.parse('2026-11-10T09:01:00Z')),'Просрочена');
});
test('notice thresholds use personal offsets and completed tasks do not alert',()=>{
 const now=new Date('2026-11-10T12:00:00Z');
 const tasks=[{id:'a',title:'a',is_completed:false,due_at:'2026-11-10T12:10:00Z'},{id:'b',is_completed:true,due_at:'2026-11-10T11:00:00Z'}];
 assert.equal(deriveNotices(tasks,[],{task_offsets:[15],event_offsets:[]},now).length,1);
 assert.equal(deriveNotices(tasks,[],{task_offsets:[5],event_offsets:[]},now).length,0);
});
