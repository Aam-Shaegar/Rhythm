"""Run against an isolated local backend. Never point this at production.
RHYTHM_TEST_API defaults to http://127.0.0.1:8089/api/v1.
RHYTHM_TEST_DB_CONTAINER defaults to rhythm-check-db.
"""
import datetime as dt
import http.cookiejar
import json
import os
import subprocess
import time
import urllib.error
import urllib.request
import uuid

base = os.environ.get('RHYTHM_TEST_API', 'http://127.0.0.1:8089/api/v1')
assert base.startswith(('http://127.0.0.1:', 'http://localhost:')), 'Use an isolated local backend'
jar = http.cookiejar.CookieJar()
op = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))
access = ''

def req(path, body=None, method=None):
    headers = {'Content-Type': 'application/json'}
    if access:
        headers['Authorization'] = 'Bearer ' + access
    data = json.dumps(body).encode() if body is not None else None
    with op.open(urllib.request.Request(base + path, data=data, headers=headers, method=method), timeout=15) as res:
        return json.load(res) if res.status != 204 else None

def sql(query):
    return subprocess.check_output(['docker', 'exec', os.environ.get('RHYTHM_TEST_DB_CONTAINER', 'rhythm-check-db'), 'psql', '-U', 'postgres', '-d', 'rhythm_check', '-At', '-v', 'ON_ERROR_STOP=1', '-c', query], text=True).strip()

def expect_error(status, path, body, method):
    try:
        req(path, body, method)
        raise AssertionError('Expected HTTP error')
    except urllib.error.HTTPError as e:
        assert e.code == status, (e.code, e.read())

name = 'check' + uuid.uuid4().hex[:12]
password = 'Local-test-password-123'
reg = req('/auth/register', {'username': name, 'email': name + '@example.test', 'password': password})
access = reg['access_token']
assert next(iter(jar)).expires - time.time() >= 14 * 86400
for _ in range(2):
    access = req('/auth/refresh', {})['access_token']
assert req('/users/me')['id'] == reg['user']['id']
assert req('/reminders/settings') == {'event_offsets': [60, 0], 'task_offsets': [60, 0]}
settings = {'event_offsets': [60, 30, 15, 5, 0], 'task_offsets': [15, 0]}
assert req('/reminders/settings', settings, 'PATCH') == settings
assert req('/reminders/settings') == settings
expect_error(400, '/reminders/settings', {'task_offsets': [1, 2, 3, 4, 5, 6]}, 'PATCH')
expect_error(400, '/reminders/settings', {'event_offsets': [15, 15]}, 'PATCH')
expect_error(400, '/reminders/settings', {'task_offsets': [-1]}, 'PATCH')
now = dt.datetime.now(dt.timezone.utc)
iso = lambda v: v.isoformat(timespec='seconds').replace('+00:00', 'Z')
future = req('/tasks', {'title': 'Future task', 'due_at': iso(now + dt.timedelta(hours=3))})
event = req('/events', {'title': 'Future event', 'start_at': iso(now + dt.timedelta(hours=4)), 'end_at': iso(now + dt.timedelta(hours=5))})
assert sql(f"SELECT count(*) FROM reminders WHERE entity_id='{future['id']}'") == '2'
assert sql(f"SELECT count(*) FROM reminders WHERE entity_id='{event['id']}'") == '5'
req('/reminders/settings', {'task_offsets': [120, 60, 30, 15, 0]}, 'PATCH')
assert sql(f"SELECT count(*) FROM reminders WHERE entity_id='{future['id']}'") == '5'
assert req('/reminders/settings')['event_offsets'] == settings['event_offsets']
late = req('/tasks', {'title': 'Late task', 'due_at': iso(now - dt.timedelta(hours=2, minutes=10))})
for _ in range(15):
    if sql(f"SELECT count(*) FROM reminders WHERE entity_id='{late['id']}' AND is_overdue") == '1': break
    time.sleep(1)
else: raise AssertionError('Worker did not queue overdue reminder')
assert sql(f"SELECT count(*) FROM reminders WHERE entity_id='{late['id']}' AND is_overdue") == '1'
completed = req('/tasks/' + late['id'] + '/complete', {}, 'PATCH')
assert completed['completed_late'] and completed['is_overdue']
assert sql(f"SELECT count(*) FROM reminders WHERE entity_id='{late['id']}'") == '0'
report = req('/reports/daily?date=' + now.strftime('%Y-%m-%d'))
assert report['completed_late'] == 1 and report['completed_on_time'] == 0, report
items = req('/tasks')
assert [t['id'] for t in items] == [future['id'], late['id']], items
# Completing through PATCH must not re-schedule reminders either.
req('/tasks/' + future['id'], {'is_completed': True}, 'PATCH')
assert sql(f"SELECT count(*) FROM reminders WHERE entity_id='{future['id']}'") == '0'
req('/auth/logout', {}, 'POST')
expect_error(401, '/auth/refresh', {}, 'POST')
print('PASS: startup, cookie session, settings persistence/limits, rescheduling, overdue worker, late reports, sorting, completion, logout')
