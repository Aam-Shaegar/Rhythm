# Деплой «Ритма» на VDS (prod runbook)

Архитектура: `caddy` (:80/:443) → `frontend` (:80, статика) и `backend:8080`
(`POST /api/*`). Фронт ходит в API **того же origin** (`/api/v1`), поэтому
в проде CORS не задействован.

## 0. TLS: что делать, если сертификата нет

Покупать ничего не нужно. Два рабочих варианта:

**A. Свой домен (рекомендуется).** Направьте A-запись домена на IP VDS,
откройте порты 80/443 и укажите в `.env`:

```env
SITE_ADDR=rhythm.example.com
CORS_ALLOWED_ORIGINS=https://rhythm.example.com
SECURE_REFRESH_COOKIE=true
```

Caddy сам выпустит бесплатный сертификат Let's Encrypt и будет его
продлевать. Проверка: замок в браузере,
`curl -s https://rhythm.example.com/healthz` → `{"status":"ok"}`.

**B. Домена нет, только IP.** Let's Encrypt не выдаёт сертификаты на
голый IP. Варианты:
- завести домен (обычно дёшево/бесплатно на первый год) → вариант A;
- Cloudflare Tunnel: `cloudflared tunnel --url http://localhost:80` —
  получаете публичный `https://…` без открытых портов и без сертификатов;
- пока нет ни того ни другого — `SITE_ADDR=:80`, обычный HTTP **только**
  для закрытого контура (тесты, VPN). Открытый прод без TLS — нельзя:
  токены и cookie будут идти открытым текстом (ТЗ S.4).

## 1. Подготовка сервера

```bash
git clone <repo> && cd Rhytm
cp .env.example .env
make secrets          # сгенерировать JWT-секреты в .env
# вручную в .env:
#   POSTGRES_PASSWORD=<длинный случайный>
#   SITE_ADDR=<домен или :80>
#   CORS_ALLOWED_ORIGINS=https://<домен>
#   SECURE_REFRESH_COOKIE=true   # только если есть HTTPS
#   LOGGER_LEVEL=INFO
```

## 2. Запуск

```bash
docker compose up -d --build
docker compose ps   # все healthy, backup сделал первый архив
```

Проверки (Caddy наружу отдаёт `/api/*`, `/healthz` и фронт):

```bash
curl -s http://localhost/healthz      # {"status":"ok"}
curl -s http://localhost/config.js     # window.__ENV__ = { API_URL: '' }
curl -s http://localhost/api/v1/users/me -H 'Authorization: Bearer x'  # 401
docker compose exec backup ls -la /backups   # rhytm-*.sql.gz
```

Фронт: открыть `http(s)://<домен>/`, зарегистрироваться, создать
задачу/событие, проверить отчёт и напоминания.

## 3. Бэкапы (ТЗ R.5)

Сервис `backup` делает `pg_dump` раз в сутки в volume `backup_data`
(`BACKUP_RETENTION_DAYS`, по умолчанию 7). Проверка и восстановление:

```bash
docker compose exec backup ls -la /backups
docker cp $(docker compose ps -q backup):/backups/rhytm-<дата>.sql.gz ./
make backup-restore file=rhytm-<дата>.sql.gz
```

## 4. Обновление

```bash
git pull
docker compose up -d --build   # миграции применятся сами (сервис migrate)
docker compose exec backend wget -q -O - http://localhost:8080/healthz
```

### Если VDS слабый и сборка идёт минутами

Первая сборка Go (~2 мин) — это нормально и разово: компилируются все
зависимости. Дальше работают BuildKit-кэши в `Dockerfile.backend`, и
пересборки занимают секунды. Если сервер совсем не тянет (мало RAM/CPU),
есть запасной путь — собрать дома и залить готовый бинарь:

```bash
# дома:
make backend-build
make backend-sync VDS=deploy@45.8.248.97:~/Rhythm
# на VDS в .env одна строка:
#   BACKEND_DOCKERFILE=Dockerfile.backend.runtime
# дальше обычный docker compose up -d --build (копирование бинаря — секунды)
```

По умолчанию этот путь не нужен — держите многостадийную сборку.

## 5. Что осознанно осталось за скобками

- R.2 (1000 concurrent): нагрузочно не тестировалось — перед ростом
  прогнать k6 vegeta-сценарий на логин/создание задачи/отчёт.
- Метрики/алерты: следующий шаг — `/metrics` + внешний мониторинг
  аптайма `/healthz`.
- Логи: ротация docker-логов настроена (`max-size 10m`), файловые логи
  backend (`LOGGER_FOLDER`) эфемерны — при разборе инцидентов снимать
  через `docker compose logs`.

## 6. Два приложения на одном VDS (Ритм + GoChat)

Порт 80/443 слушает только общий Caddy из этого проекта. GoChat свой
порт 80 не публикует — его nginx виден Caddy по имени `gochat-nginx`
через общую docker-сеть `edge`. Маршрутизация — по имени сайта.

Предусловия на VDS: оба имени резолвятся в IP сервера
(`nslookup <имя>` → IP VDS), открыты входящие 80/443.

```bash
docker network create edge   # один раз; ignore, если уже есть

# GoChat: отдать :80 (конфиг уже готов в репозитории GoChat)
cd ~/projects/golang/GoChat
docker compose up -d --build

# Rhytm: два сайта (конфиг уже готов: Caddyfile + GOCHAT_ADDR)
cd ~/projects/golang/Rhytm
docker compose up -d --build
```

Нужные значения в `.env` (Rhytm, уезжают на VDS через git):

```env
SITE_ADDR=rhythm-tesler.duckdns.org
GOCHAT_ADDR=gochat-tesler.duckdns.org
CORS_ALLOWED_ORIGINS=http://localhost:5173,http://localhost:3000,https://rhythm-tesler.duckdns.org,https://gochat-tesler.duckdns.org
SECURE_REFRESH_COOKIE=true
```

В GoChat `.env` **на сервере** (файл не в git — править прямо на VDS):

```env
CORS_ALLOWED_ORIGINS=https://gochat-tesler.duckdns.org
SECURE_REFRESH_COOKIE=true
```

Проверки:

```bash
curl -s https://rhythm-tesler.duckdns.org/healthz   # {"status":"ok"}
curl -s -o /dev/null -w '%{http_code}\n' https://gochat-tesler.duckdns.org/  # 200
# Браузер: оба замка зелёные. GoChat: вход + сообщение (проверка wss).
# Ритм: вход, задача, событие, отчёт.
```

Откат: если что-то пошло не так — в GoChat compose вернуть
`ports: ["80:80"]` у nginx и `docker compose up -d` (GoChat снова один
на :80), а Caddy Ритма остановить. Данные при этом не трогаем:
у каждого проекта свои volumes.
