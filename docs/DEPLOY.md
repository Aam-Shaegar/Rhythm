# Production-деплой

Для первого знакомства используйте [локальный стек](../README.md#запуск-за-несколько-команд). Этот документ описывает production-файл `docker-compose.yml` и отдельный процесс выпуска через git.

## Устройство

Caddy принимает HTTP/HTTPS и направляет `/api/*` и `/healthz` в Go-backend, остальные запросы — в nginx со статическим frontend. PostgreSQL хранит данные, Redis используется приложением, `migrate` применяет схему до запуска backend. `backup` создаёт ежедневные архивы PostgreSQL.

Production Compose по умолчанию использует `Dockerfile.backend.runtime`: Linux-бинарник собирается локально и доставляется через git. Альтернатива — `BACKEND_DOCKERFILE=Dockerfile.backend` в серверном `.env`, тогда backend собирается из исходников на сервере.

## Первый запуск

```bash
git clone https://github.com/Aam-Shaegar/Rhythm.git
cd Rhythm
python3 scripts/init_env.py
```

Настройте **на сервере** `.env`:

```dotenv
SITE_ADDR=rhythm.example.com
CORS_ALLOWED_ORIGINS=https://rhythm.example.com
SECURE_REFRESH_COOKIE=true
LOGGER_LEVEL=INFO
# Необязательно: домен второго приложения, обслуживаемого общим Caddy.
GOCHAT_ADDR=
```

JWT-секреты и пароль PostgreSQL сгенерированы `init_env.py`. Для существующей БД сохраните её действующие реквизиты. Изменение пароля в `.env` само по себе не меняет пароль уже созданного PostgreSQL-пользователя.

Для HTTPS домен должен указывать на сервер, входящие порты 80/443 — быть открыты. Caddy управляет сертификатом. Web Push требует HTTPS и настроенных `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_SUBJECT`; пару можно получить через `make vapid`.

Production-конфигурация поддерживает общий прокси со вторым приложением GoChat и использует внешнюю сеть `edge`. Создайте её один раз, если ещё нет:

```bash
docker network inspect edge >/dev/null 2>&1 || docker network create edge
sha256sum -c build/rhytm.sha256
docker compose config -q
docker compose up -d --build
docker compose ps
curl -fsS https://rhythm.example.com/healthz
```

`/healthz` проверяет доступность процесса. Дополнительно проверьте вход, создание задачи и отчёт. Не публикуйте сгенерированный вывод конфигурации с секретами; для валидации используйте `config -q`.

## Подготовка выпуска

На машине разработки после изменений backend:

```bash
npm ci --prefix frontend
bash scripts/check_ml.sh
make backend-build
git add build/rhytm build/rhytm.sha256
# Закоммитьте исходники, миграции, бинарник и checksum вместе, затем git push.
```

`build/rhytm` — статический `linux/amd64` бинарник. Для ARM-сервера потребуется отдельная сборка и соответствующий образ. Frontend собирается на сервере из `package-lock.json` командой `npm ci`.

## Обновление сервера

Сначала сохраните текущий commit, серверный `.env`, предыдущий бинарник/образы и резервную копию БД. Конфигурация не доставляется через git.

**Первый переход со старой версии, где `.env` был tracked:** `git pull` может удалить файл с диска. До обновления сохраните оба `.env` вне каталога checkout с правами 600; после pull восстановите серверные значения. Не заменяйте их шаблоном. В новых версиях `.env` игнорируется.

```bash
# Из каталога проекта; архив хранить вне checkout.
docker compose exec -T postgres sh -c \
  'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB"' | gzip > ../rhythm-before-update.sql.gz
gzip -t ../rhythm-before-update.sql.gz

git pull --ff-only
sha256sum -c build/rhytm.sha256
docker compose config -q
docker compose build backend frontend
docker compose run --rm -T migrate
docker compose up -d
docker compose ps
docker compose logs --since 5m --tail 80 backend
curl -fsS https://rhythm.example.com/healthz
```

Не меняйте ветку сервера вслепую: сначала проверьте её историю и рабочее дерево. Не включайте `down -v` в обновление — эта команда удаляет volumes с данными. Миграции запускаются явно, чтобы ошибка схемы остановила обновление до перезапуска приложения.

## Резервные копии и восстановление

Сервис `backup` сохраняет архивы в volume `backup_data`; срок хранения задаётся `BACKUP_RETENTION_DAYS` (по умолчанию 7). Переносите копии за пределы сервера и периодически проверяйте восстановление в отдельной БД.

```bash
docker compose exec backup ls -lh /backups
# Получить конкретный архив:
docker cp "$(docker compose ps -q backup)":/backups/<archive.sql.gz> ./
```

`make backup-restore file=<archive.sql.gz>` загружает архив в настроенную БД. Это изменяет данные: выполняйте только при осознанном восстановлении, с остановленной записью и отдельной актуальной копией.

## Откат и ограничения

Для отката приложения используйте сохранённые commit/образы, совместимые с текущей схемой. Не запускайте миграции down автоматически: они могут удалять данные. Миграция ML №11 аддитивна; старую версию приложения можно вернуть, оставив добавленные таблицы. Её down удаляет ML-профили и снимки.

Нагрузочные пределы production не измерены. Мониторинг, метрики, алерты и измерение качества ML на реальной временной выборке остаются отдельными задачами. Логи ограничены Docker-ротацией; файловые логи backend внутри контейнера не являются долговечным хранилищем.
