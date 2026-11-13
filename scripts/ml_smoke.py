"""End-to-end ML checks. Only an isolated localhost backend/database is accepted."""
import argparse
from concurrent.futures import ThreadPoolExecutor
import datetime as dt
import http.cookiejar
import json
import os
from pathlib import Path
import subprocess
import time
import urllib.request
import urllib.error
import uuid

BASE = os.environ.get('RHYTHM_ML_API', 'http://127.0.0.1:8092/api/v1')
assert BASE.startswith(('http://127.0.0.1:', 'http://localhost:')), 'Use an isolated local backend'
CONTAINER = os.environ.get('RHYTHM_ML_DB', 'rhythm-ml-db')
PASSWORD = 'Ml-local-check-123'
ISO = lambda d: d.isoformat(timespec='seconds').replace('+00:00', 'Z')
NOW = lambda: dt.datetime.now(dt.timezone.utc)

def sql(query, expect_failure=False):
    result = subprocess.run(['docker', 'exec', CONTAINER, 'psql', '-U', 'postgres', '-d', 'rhythm_check', '-At', '-v', 'ON_ERROR_STOP=1', '-c', query], capture_output=True, text=True)
    if expect_failure:
        assert result.returncode != 0 and 'foreign key' in result.stderr, result.stderr
    else:
        assert result.returncode == 0, result.stderr
    return result.stdout.strip()

class Client:
    def __init__(self, token=''):
        self.token = token
        self.opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
    def req(self, path, body=None, method=None):
        headers = {'Content-Type': 'application/json'}
        if self.token:
            headers['Authorization'] = 'Bearer ' + self.token
        request = urllib.request.Request(BASE + path, data=json.dumps(body).encode() if body is not None else None, method=method, headers=headers)
        with self.opener.open(request, timeout=20) as response:
            return json.load(response) if response.status != 204 else None
    def register(self, prefix):
        name = prefix + uuid.uuid4().hex[:12]
        data = self.req('/auth/register', {'username': name, 'email': name + '@example.test', 'password': PASSWORD})
        self.token = data['access_token']
        self.uid = data['user']['id']
        self.name = name
        return self
    def error(self, status, path, body=None, method=None):
        try:
            self.req(path, body, method)
            raise AssertionError('expected HTTP error')
        except urllib.error.HTTPError as e:
            assert e.code == status, (e.code, e.read())
    def task(self, title, hours=24, importance=2, minutes=30, due=None):
        return self.req('/tasks', {'title': title, 'description': title, 'due_at': ISO(due or NOW() + dt.timedelta(hours=hours)), 'importance': importance, 'estimated_minutes': minutes})
    def ranking(self):
        return self.req('/tasks/priorities')

def profile(uid):
    uuid.UUID(uid)
    return json.loads(sql(f"SELECT json_build_object('hash',dataset_hash,'delta',delta,'samples',samples,'trained_at',trained_at) FROM user_priority_profiles WHERE user_id='{uid}'"))

def run(state_path):
    a, b = Client().register('mlA'), Client().register('mlB')
    Client().error(401, '/tasks/priorities')
    empty = a.ranking()
    assert empty['personal_samples'] == 0 and empty['items'] == []
    study = a.task('Подготовить экзамен по математике', hours=168, importance=1, minutes=60)
    finance = a.task('Оплатить налог и счет банка', hours=2, importance=3, minutes=90)
    foreign = b.task('Купить продукты молоко магазин', hours=72)
    a.error(400, '/tasks', {'title':'Invalid','due_at':ISO(NOW()+dt.timedelta(days=1)),'importance':4})
    a.error(400, '/tasks', {'title':'Invalid','due_at':ISO(NOW()+dt.timedelta(days=1)),'estimated_minutes':0})
    a.error(404, '/tasks/' + foreign['id'])
    ranking = a.ranking()
    assert ranking['items'][0]['task_id'] == finance['id'], ranking
    assert ranking['items'][0]['category'] == 'finance', ranking['items'][0]
    assert ranking['items'][0]['level'] == 'high'
    assert len(ranking['recommendations']) >= 1 and ranking['personal_samples'] == 0
    assert all(item['task_id'] != foreign['id'] for item in a.req('/tasks/priorities?user_id='+b.uid)['items'])
    before = profile(a.uid)
    # Separate HTTP clients exercise concurrent DB transactions, not a client-side cache.
    with ThreadPoolExecutor(max_workers=8) as executor:
        list(executor.map(lambda _: Client(a.token).ranking(), range(8)))
    assert profile(a.uid) == before, 'repeat reads retrained the model'
    assert sql(f"SELECT count(*) FROM task_priority_snapshots WHERE user_id='{a.uid}'") == '2'
    features_before = sql(f"SELECT features FROM task_priority_snapshots WHERE task_id='{finance['id']}'")
    a.req('/tasks/' + finance['id'] + '/complete', {}, 'PATCH')
    assert a.ranking()['personal_samples'] == 1
    assert features_before == sql(f"SELECT features FROM task_priority_snapshots WHERE task_id='{finance['id']}'"), 'completion replaced pre-outcome features'
    assert b.ranking()['personal_samples'] == 0
    assert any(v != 0 for v in profile(a.uid)['delta']) and all(v == 0 for v in profile(b.uid)['delta'])
    a.req('/tasks/'+finance['id'], {'is_completed':False}, 'PATCH')
    assert a.ranking()['personal_samples'] == 0, 'reopening did not retract a label'
    deadline = NOW() + dt.timedelta(seconds=8)
    late = a.task('Убрать квартиру и помыть посуду', due=deadline)
    assert sql(f"SELECT count(*) FROM task_priority_snapshots WHERE task_id='{late['id']}'") == '1'
    for _ in range(24):
        current = a.ranking()
        if current['personal_samples'] == 1:
            break
        time.sleep(.5)
    else:
        raise AssertionError('deadline outcome was not learned')
    prediction = next(item for item in current['items'] if item['task_id'] == late['id'])
    assert prediction['risk'] == 1 and prediction['level'] == 'high'
    learned = profile(a.uid)
    a.req('/tasks/'+late['id']+'/complete', {}, 'PATCH')
    assert a.ranking()['personal_samples'] == 1 and profile(a.uid) == learned, 'same overdue task was learned twice'
    a.req('/tasks/'+late['id'], {'is_completed':False,'due_at':ISO(NOW()+dt.timedelta(days=3))}, 'PATCH')
    assert a.ranking()['personal_samples'] == 0, 'deadline transfer did not retract stale outcome'
    a.req('/tasks/'+late['id'], None, 'DELETE')
    assert sql(f"SELECT count(*) FROM task_priority_snapshots WHERE task_id='{late['id']}'") == '0'
    past = a.task('Историческая задача без снимка', hours=-24)
    assert a.ranking()['personal_samples'] == 0, 'historical outcome leaked into training'
    # Ownership is enforced by a composite FK even below the API layer.
    sql(f"INSERT INTO task_priority_snapshots(task_id,user_id,revision,model_version,features,observed_at,due_at) VALUES('{past['id']}','{b.uid}','{'0'*64}','test','{json.dumps([0]*18)}',now()-interval '2 days',now()-interval '1 day')", expect_failure=True)
    a.req('/tasks/'+study['id']+'/complete', {}, 'PATCH')
    # A fresh on-time completion remains in personal history for restart verification.
    final = a.ranking()
    assert final['personal_samples'] == 1
    assert all(item['task_id'] != study['id'] for item in final['items'])
    state = {'token':a.token,'uid':a.uid,'name':a.name,'profile':profile(a.uid),'samples':final['personal_samples']}
    state_path.write_text(json.dumps(state));state_path.chmod(0o600)
    print('PASS: live ML API, category/ranking, validation, tenant isolation, concurrent reads, pre-outcome snapshots, on-time/overdue learning, retraction, deletion and FK ownership')
    print('Test account for local UI: '+a.name)

def resume(state_path):
    state=json.loads(state_path.read_text());client=Client(state['token']);ranking=client.ranking()
    assert ranking['personal_samples']==state['samples']
    assert profile(state['uid'])==state['profile'], 'restart changed weights or retrained unchanged history'
    print('PASS: personal weights, snapshots and model version survive backend restart')

if __name__ == '__main__':
    parser=argparse.ArgumentParser();parser.add_argument('--resume',action='store_true');parser.add_argument('--state',default='/private/tmp/rhythm-ml-smoke-state.json');args=parser.parse_args()
    (resume if args.resume else run)(Path(args.state))
