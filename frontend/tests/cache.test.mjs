import test from 'node:test';
import assert from 'node:assert/strict';
const values = new Map();
globalThis.localStorage = { getItem:k=>values.get(k)??null,setItem:(k,v)=>values.set(k,v),removeItem:k=>values.delete(k) };
globalThis.window = {__ENV__:{API_URL:'/api/v1'},dispatchEvent(){}};
const {apiFetch,setTokens,clearTokens}=await import('../src/api/client.ts');
const json = v=>new Response(JSON.stringify(v));
test('deduplicates reads, clones cached results, invalidates writes and account changes',async()=>{
 let calls=0;
 globalThis.fetch=async()=>{ calls++;await new Promise(r=>setTimeout(r,5));return json([{id:calls}]); };
 setTokens('user-a');
 const [a,b]=await Promise.all([apiFetch('/tasks'),apiFetch('/tasks')]);
 assert.equal(calls,1);a[0].id=99;assert.equal(b[0].id,1);
 assert.equal((await apiFetch('/tasks'))[0].id,1);assert.equal(calls,1);
 await apiFetch('/tasks',{method:'POST',body:'{}'});
 await apiFetch('/tasks');assert.equal(calls,3);
 clearTokens();setTokens('user-b');await apiFetch('/tasks');assert.equal(calls,4);
});
