import test from 'node:test';
import assert from 'node:assert/strict';
import {comparePriorityTasks, compareTasks} from '../src/tasks.ts';

test('priority sorting changes active order while completed tasks remain last',()=>{
 const tasks=[{id:'urgent',is_completed:false,due_at:'2026-11-13T09:00:00Z'},{id:'risk',is_completed:false,due_at:'2026-11-13T10:00:00Z'},{id:'done',is_completed:true,due_at:'2026-11-12T09:00:00Z'}];
 const scores=new Map([['urgent',{score:65}],['risk',{score:90}],['done',{score:100}]]);
 assert.deepEqual([...tasks].sort((a,b)=>comparePriorityTasks(a,b,scores)).map(t=>t.id),['risk','urgent','done']);
 assert.deepEqual([...tasks].sort(compareTasks).map(t=>t.id),['urgent','risk','done']);
});
test('equal or unavailable priority scores preserve stable deadline ordering',()=>{
 const tasks=[{id:'b',is_completed:false,due_at:'2026-11-13T10:00:00Z'},{id:'a',is_completed:false,due_at:'2026-11-13T10:00:00Z'},{id:'early',is_completed:false,due_at:'2026-11-13T09:00:00Z'}];
 for(const scores of [new Map(),new Map(tasks.map(t=>[t.id,{score:50}]))]) assert.deepEqual([...tasks].sort((a,b)=>comparePriorityTasks(a,b,scores)).map(t=>t.id),['early','a','b']);
});
