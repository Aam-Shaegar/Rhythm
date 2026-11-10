import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import vm from 'node:vm';

test('worker caches hashed assets, excludes private API, and serves shell offline',async()=>{
 const handlers={};const stored=new Map([['/index.html',new Response('<html>offline shell</html>')]]);let network=0;
 const cache={match:async req=>stored.get(typeof req==='string'?req:req.url)?.clone(),put:async(req,res)=>stored.set(req.url,res),addAll:async()=>{}};
 const context={URL,Response,AbortSignal,console,caches:{open:async()=>cache,keys:async()=>[],delete:async()=>true},fetch:async()=>{network++;return new Response('asset')},self:{location:{origin:'https://rhythm.test'},addEventListener:(name,fn)=>handlers[name]=fn,skipWaiting:async()=>{},clients:{claim:async()=>{}}}};
 vm.runInNewContext(readFileSync(new URL('../public/sw.js',import.meta.url),'utf8'),context);
 async function get(path,mode='cors'){
  let result;
  handlers.fetch({request:{url:'https://rhythm.test'+path,method:'GET',mode},respondWith:p=>result=p});
  return result;
 }
 assert.equal(await get('/api/v1/tasks'),undefined);assert.equal(await get('/config.js'),undefined);
 assert.equal(await (await get('/assets/forest-hash.webp')).text(),'asset');
 assert.equal(await (await get('/assets/forest-hash.webp')).text(),'asset');assert.equal(network,1);
 context.fetch=async()=>{throw new TypeError('offline')};
 assert.match(await (await get('/?tab=tasks','navigate')).text(),/offline shell/);
});
