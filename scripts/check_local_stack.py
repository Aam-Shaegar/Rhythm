#!/usr/bin/env python3
"""Smoke-test only the disposable local Compose stack, never a remote server."""
import datetime as dt
import http.cookiejar
import json
import re
import time
import urllib.error
import urllib.request
import uuid

BASE = 'http://127.0.0.1:8088'
jar = http.cookiejar.CookieJar()
opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))
access = ''


def api(path, body=None, method=None):
    headers = {'Content-Type': 'application/json'}
    if access:
        headers['Authorization'] = 'Bearer ' + access
    request = urllib.request.Request(
        BASE + '/api/v1' + path,
        data=json.dumps(body).encode() if body is not None else None,
        headers=headers, method=method,
    )
    with opener.open(request, timeout=20) as response:
        return json.load(response) if response.status != 204 else None


for attempt in range(30):
    try:
        with opener.open(BASE + '/healthz', timeout=3) as response:
            assert json.load(response)['status'] == 'ok'
        break
    except (urllib.error.URLError, TimeoutError):
        if attempt == 29:
            raise
        time.sleep(1)
with opener.open(BASE, timeout=20) as response:
    html = response.read().decode()
assets = re.findall(r'(?:src|href)="([^"]+\.(?:js|css))"', html)
assert any(path.startswith('/assets/') for path in assets)
for path in assets:
    with opener.open(BASE + path, timeout=20) as response:
        assert response.status == 200
with opener.open(BASE + '/sw.js', timeout=20) as response:
    assert "const PRECACHE = [];" not in response.read().decode()
try:
    api('/tasks/priorities')
except urllib.error.HTTPError as error:
    assert error.code == 401
else:
    raise AssertionError('ML endpoint must require authentication')

username = 'stack' + uuid.uuid4().hex[:12]
password = uuid.uuid4().hex + 'A1!'
session = api('/auth/register', {
    'username': username, 'email': username + '@example.test', 'password': password,
})
access = session['access_token']
assert any(cookie.name == 'refresh_token' for cookie in jar)
task = api('/tasks', {
    'title': 'Оплатить счёт', 'description': 'Проверка локального запуска',
    'due_at': (dt.datetime.now(dt.timezone.utc) + dt.timedelta(hours=2)).isoformat(),
    'importance': 3, 'estimated_minutes': 15,
})
try:
    priorities = api('/tasks/priorities')
    assert any(item['task_id'] == task['id'] for item in priorities['items'])
    assert api('/tasks/' + task['id'] + '/complete', {}, 'PATCH')['is_completed']
    assert api('/auth/refresh', {}, 'POST')['access_token']
    api('/auth/logout', {}, 'POST')
    try:
        api('/auth/refresh', {}, 'POST')
    except urllib.error.HTTPError as error:
        assert error.code == 401
    else:
        raise AssertionError('Logout must revoke the refresh session')
finally:
    # Access tokens remain valid until their short TTL; remove our test task.
    api('/tasks/' + task['id'], method='DELETE')
print('PASS: local site/assets/PWA, protected API, registration, task, ML, refresh and logout')
