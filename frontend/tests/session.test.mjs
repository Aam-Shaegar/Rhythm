import test from 'node:test';
import assert from 'node:assert/strict';
const values = new Map();
globalThis.localStorage = { getItem: k => values.get(k) ?? null, setItem: (k,v) => values.set(k,v), removeItem: k => values.delete(k) };
globalThis.window = { __ENV__: { API_URL: '/api/v1' }, dispatchEvent() {} };
const { apiFetch, setTokens, getAccessToken, clearTokens, refreshAccessToken } = await import('../src/api/client.ts');
const json = (body, status = 200) => new Response(JSON.stringify(body), { status });

test('login cookie can refresh with no localStorage refresh token; concurrent requests refresh once', async () => {
 values.clear(); setTokens('expired');
 let refreshes = 0;
 globalThis.fetch = async (url, init) => {
  if (url.endsWith('/auth/refresh')) {
   refreshes++;
   assert.equal(init.credentials, 'include');
   await new Promise(r => setTimeout(r, 5));
   return json({ access_token: 'fresh' });
  }
  return init.headers.Authorization === 'Bearer fresh' ? json({ id: 'user' }) : json({}, 401);
 };
 const result = await Promise.all([apiFetch('/users/me'), apiFetch('/tasks'), apiFetch('/events')]);
 assert.equal(result.length, 3); assert.equal(refreshes, 1); assert.equal(getAccessToken(), 'fresh');
});

test('network and server failures preserve the session', async () => {
 setTokens('expired');
 globalThis.fetch = async url => url.endsWith('/auth/refresh') ? json({}, 503) : json({}, 401);
 await assert.rejects(apiFetch('/users/me'));
 assert.equal(getAccessToken(), 'expired');
 globalThis.fetch = async () => { throw new TypeError('offline'); };
 await assert.rejects(refreshAccessToken());
 assert.equal(getAccessToken(), 'expired');
});

test('cookie bootstraps when the access token is missing', async () => {
 clearTokens();
 globalThis.fetch = async (url, init) => url.endsWith('/auth/refresh') ? json({access_token:'cookie-access'}) : init.headers.Authorization ? json({id:'u'}) : json({},401);
 assert.deepEqual(await apiFetch('/users/me'), {id:'u'});
});

test('a definitively expired refresh clears stored credentials', async () => {
 setTokens('expired');
 globalThis.fetch = async () => json({}, 401);
 await assert.rejects(apiFetch('/users/me'));
 assert.equal(getAccessToken(), null);
});
