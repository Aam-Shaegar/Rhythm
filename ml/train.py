"""Reproducible synthetic baseline, standard library only. No user data is read."""
import argparse
import collections
import hashlib
import json
import math
import random
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CATEGORIES = ['work', 'study', 'finance', 'home', 'health', 'personal', 'other']
FEATURES = ['bias', 'remaining_log', 'lead_log', 'effort', 'active_load', 'soon_load', 'description_missing', 'text_length', 'recurring', 'importance', 'due_weekend'] + ['cat_' + c for c in CATEGORIES]
# Topic lexicons seed synthetic examples; production classification uses learned token likelihoods.
TOPICS = {
 'work': ['отчет проект клиент договор совещание', 'презентация коллеги работа встреча', 'разработать код сервер исправить баг', 'письмо заказчик релиз команда', 'report project client meeting email deploy work'],
 'study': ['экзамен учеба лекция конспект', 'курсовая диплом университет семинар', 'домашнее задание математика урок', 'прочитать учебник решить упражнение', 'study homework exam lecture lesson course'],
 'finance': ['оплатить счет налог банк', 'бюджет кредит платеж деньги', 'страховка квитанция коммунальные расходы', 'подать декларацию перевод аренда', 'pay invoice tax budget bank loan money'],
 'home': ['уборка квартира дом кухня', 'купить продукты магазин молоко', 'постирать одежда посуда ремонт', 'вынести мусор приготовить ужин', 'home groceries clean laundry repair kitchen'],
 'health': ['врач прием анализы поликлиника', 'тренировка спорт пробежка здоровье', 'записаться стоматолог обследование', 'лекарство аптека витамины терапевт', 'health doctor dentist medicine workout exercise'],
 'personal': ['день рождения подарок друзья', 'позвонить мама семья родители', 'отпуск путешествие билеты поездка', 'хобби гитара прогулка фотография', 'personal birthday family friends travel hobby'],
 'other': ['разобрать заметки идеи список', 'проверить варианты информация вопрос', 'уточнить детали записать напоминание', 'план выбрать вариант продолжить', 'misc check options notes ideas reminder'],
}
TOPICS['work'] += ['резюме собеседование вакансия карьера', 'спринт тестирование документация ревью макет']
TOPICS['study'] += ['лабораторная лабораторную лабу лабы домашка домашку', 'сдать зачет зачёт контрольная реферат университет']
TOPICS['finance'] += ['выплатить долг ипотека подписка зарплата', 'заплатить электричество пополнить баланс коммуналка']
TOPICS['home'] += ['убрать помыть пол полы пылесос', 'купить хлеб овощи фрукты яйца', 'покормить кошку собаку питомец корм']
TOPICS['health'] += ['больница окулист обследование кардиолог', 'спортзал фитнес бассейн йога зарядка']
TOPICS['personal'] += ['поздравить друга встретиться родственники', 'кино театр концерт музей выходные']

TEMPLATES = [('Нужно {}', 'План: {}. Подготовить всё необходимое.'), ('Завершить: {}', 'Важные детали: {}'), ('На этой неделе — {}', 'Не забыть: {}'), ('Запланировать {}', '{}; проверить результат'), ('{}', 'Заняться: {}'), ('Подготовить {}', 'Следующий шаг: {}')]

def tokens(title, description=''):
    words = re.findall(r'[^\W_]+', (title + ' ' + title + ' ' + description).lower().replace('ё', 'е'), flags=re.UNICODE)
    return [word[:7] for word in words if len(word) >= 3]

def text_rows(seed, count, held_out=False):
    rng = random.Random(seed)
    templates = TEMPLATES[4:] if held_out else TEMPLATES[:4]
    rows = []
    for i in range(count):
        category = CATEGORIES[i % len(CATEGORIES)]
        groups = TOPICS[category]
        words = rng.choice(groups).split()
        selected = rng.sample(words, min(len(words), rng.randint(2, 4)))
        more = rng.choice(groups).split()
        title_template, desc_template = rng.choice(templates)
        rows.append((title_template.format(' '.join(selected)), desc_template.format(' '.join(more)), category))
    rng.shuffle(rows)
    return rows

def train_category(rows):
    counts = {c: collections.Counter() for c in CATEGORIES}
    documents = collections.Counter()
    for title, desc, category in rows:
        counts[category].update(tokens(title, desc)); documents[category] += 1
    vocab = sorted(set().union(*(set(c) for c in counts.values())))
    weights = []
    for c in CATEGORIES:
        total = sum(counts[c].values()) + len(vocab)
        weights.append([round(math.log((counts[c][v] + 1) / total), 9) for v in vocab])
    return {'vocabulary': vocab, 'log_likelihood': weights, 'log_prior': [math.log(documents[c] / len(rows)) for c in CATEGORIES]}

def classify(model, title, desc=''):
    positions = {v: i for i, v in enumerate(model['vocabulary'])}
    counts = collections.Counter(tokens(title, desc))
    recognized = [(positions[w], min(n, 3)) for w, n in counts.items() if w in positions]
    if not recognized:
        return 'other', [0.] * 6 + [1.]
    scores = [model['log_prior'][c] + sum(model['log_likelihood'][c][i] * n for i, n in recognized) for c in range(7)]
    m = max(scores); exp = [math.exp(s - m) for s in scores]; total = sum(exp); probs = [v / total for v in exp]
    best = max(range(7), key=lambda i: probs[i])
    if probs[best] < .52:
        return 'other', [0.] * 6 + [1.]
    return CATEGORIES[best], probs

def sigmoid(z):
    return 1 / (1 + math.exp(-max(-30., min(30., z))))

def features(hours, lead, effort, active, soon, missing, length, recurring, importance, weekend, probs):
    norm = lambda h: math.log1p(max(0., min(720., h))) / math.log(721.)
    return [1., norm(hours), norm(lead), min(480., effort) / 480., min(20., active) / 20., min(10., soon) / 10., float(missing), min(2000., length) / 2000., float(recurring), (importance - 1) / 2., float(weekend)] + list(probs)

def risk_rows(seed, count):
    rng = random.Random(seed); rows = []
    truth = [-.2, -2.4, .7, 1.6, 1.3, 1., .4, .3, -.3, -.7, .25, -.15, .25, -.35, .4, -.25, .15, 0.]
    for _ in range(count):
        category = rng.randrange(7); probs = [.02] * 7; probs[category] = .88
        hours = 10 ** rng.uniform(-.6, 2.85); lead = max(hours, hours * rng.uniform(1, 4))
        x = features(hours, lead, rng.choice([10, 15, 30, 60, 120, 240, 480]), rng.randrange(31), rng.randrange(16), rng.random() < .3, rng.randrange(1800), rng.random() < .2, rng.randint(1, 3), rng.random() < 2/7, probs)
        probability = sigmoid(sum(a*b for a,b in zip(truth,x)))
        rows.append((x, int(rng.random() < probability)))
    return rows

def train_risk(rows, seed):
    rng = random.Random(seed); weights = [0.] * len(FEATURES); shuffled = list(rows)
    for epoch in range(24):
        rng.shuffle(shuffled); rate = .065 / (1 + epoch * .12)
        for x,y in shuffled:
            error = sigmoid(sum(w*v for w,v in zip(weights,x))) - y
            weights = [w - rate * (error*v + (.002*w if j else 0)) for j,(w,v) in enumerate(zip(weights,x))]
    return [round(w, 9) for w in weights]

def evaluate(weights, rows):
    predictions = [(sigmoid(sum(w*v for w,v in zip(weights,x))), y) for x,y in rows]
    mean = sum(y for _,y in predictions) / len(rows)
    loss = -sum(y*math.log(max(p,1e-9))+(1-y)*math.log(max(1-p,1e-9)) for p,y in predictions)/len(rows)
    brier = sum((p-y)**2 for p,y in predictions)/len(rows)
    negatives = sum(1-y for _,y in predictions); positives = len(rows)-negatives
    # AUC via average ranks, with tie handling.
    ordered = sorted(predictions); rank_sum = 0.; i = 0
    while i < len(ordered):
        j = i+1
        while j < len(ordered) and ordered[j][0] == ordered[i][0]: j += 1
        rank_sum += sum(y for _,y in ordered[i:j]) * ((i+1+j)/2)
        i = j
    auc = (rank_sum-positives*(positives+1)/2)/(positives*negatives)
    return {'log_loss': round(loss,6), 'brier': round(brier,6), 'constant_brier': round(mean*(1-mean),6), 'roc_auc': round(auc,6), 'late_rate': round(mean,6), 'samples': len(rows)}

def build():
    category = train_category(text_rows(713, 3500))
    held = text_rows(714, 1050, held_out=True)
    accuracy = sum(classify(category,t,d)[0]==c for t,d,c in held)/len(held)
    weights = train_risk(risk_rows(715, 10000), 716)
    metrics = {'scope': 'synthetic_only_not_real_world_accuracy', 'seed': 713, 'category_holdout': {'accuracy': round(accuracy,6), 'samples': len(held), 'split': 'held-out templates'}, 'risk_holdout': evaluate(weights,risk_rows(717,3000))}
    probes = [{'title':'Оплатить налог', 'description':'Проверить декларацию и счет банка'}, {'title':'Подготовить экзамен', 'description':'Повторить учебник и лекции'}, {'title':'Починить сервер', 'description':'Исправить баг в проекте'}, {'title':'zxqv 123','description':''}]
    for p in probes:
        p['category'],p['probabilities']=classify(category,p['title'],p['description'])
    feature_probe = {'title': 'Оплатить налог', 'description': 'Проверить декларацию и счет банка', 'now': '2026-11-13T12:00:00Z', 'created_at': '2026-11-13T10:00:00Z', 'due_at': '2026-11-14T18:00:00Z'}
    _, probe_probs = classify(category, feature_probe['title'], feature_probe['description'])
    feature_probe['features'] = features(30, 32, 90, 4, 2, False, len(feature_probe['title'])+len(feature_probe['description']), False, 3, True, probe_probs)
    feature_probe['risk'] = sigmoid(sum(w*x for w,x in zip(weights, feature_probe['features'])))
    version = 'priority-v1-' + hashlib.sha256(json.dumps({'text': category, 'risk': weights, 'features': FEATURES}, sort_keys=True, ensure_ascii=False).encode()).hexdigest()[:12]
    artifact = {'schema': 1, 'version': version, 'categories': CATEGORIES, 'feature_names': FEATURES, 'text': category, 'risk_weights': weights, 'probes': probes, 'metrics': metrics, 'feature_probe': feature_probe}
    return artifact, metrics

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--check',action='store_true');args=parser.parse_args()
    artifact, metrics = build()
    assert metrics['category_holdout']['accuracy'] >= .9, metrics
    assert metrics['risk_holdout']['roc_auc'] > .65 and metrics['risk_holdout']['brier'] < metrics['risk_holdout']['constant_brier'], metrics
    path=ROOT/'internal/features/priorities/model/baseline.json';data=json.dumps(artifact,ensure_ascii=False,sort_keys=True,indent=2)+'\n'
    if args.check: assert path.read_text()==data, 'baseline is not reproducible'
    else:
        path.write_text(data);(ROOT/'ml/metrics.json').write_text(json.dumps(metrics,indent=2)+'\n')
    print(json.dumps(metrics,ensure_ascii=False,indent=2));print('artifact_sha256='+hashlib.sha256(data.encode()).hexdigest())
if __name__=='__main__': main()
