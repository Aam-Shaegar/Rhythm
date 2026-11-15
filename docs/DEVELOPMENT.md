# Разработка

## Требования

- Go версии из `go.mod` (сейчас 1.25.6).
- Node.js 22.18+; frontend-тесты используют встроенный запуск TypeScript в Node.
- Python 3, без внешних пакетов для обучения модели и smoke-тестов.
- Docker с Compose v2 для PostgreSQL, Redis и интеграционных проверок.

Для знакомства с продуктом используйте [локальный Docker-стек](../README.md#запуск-за-несколько-команд): ему не нужны Go и Node на хосте.

## Проверки по отдельности

```bash
npm ci --prefix frontend
go test ./...
go test -race ./internal/features/priorities/...
go vet ./...
npm --prefix frontend test
npm --prefix frontend run build
python3 -m unittest discover -s ml -v
python3 ml/train.py --check
```

Полный набор: `bash scripts/check_ml.sh` или `make test-all`. Он создаёт собственные контейнеры со случайными портами, проверяет реальные запросы и повторяет чтение модели после перезапуска backend. Не использует production `.env` или production-БД. До запуска установите npm-зависимости.

Дополнительная проверка документированного способа установки:

```bash
make init
make local-up
python3 scripts/check_local_stack.py
make local-down
```

Smoke-тест создаёт отдельный тестовый аккаунт в локальной БД, удаляет созданную задачу и отзывает сессию. Он жёстко привязан к `127.0.0.1:8088`. Запускайте его в демонстрационном стеке.

## Изменение интерфейса

Запустите локальный стек, затем Vite с прокси к его API:

```bash
VITE_API_URL=http://localhost:8088/api/v1 npm --prefix frontend run dev
```

Откройте `http://localhost:5173`. `frontend/public/config.js` задаёт same-origin `/api/v1`, а Vite проксирует запросы в backend через локальный gateway. Настройка `VITE_API_URL` для прокси передаётся в окружении процесса. Подробности frontend — в [frontend/README.md](../frontend/README.md).

## Работа с моделью

```bash
python3 ml/train.py          # переобучить базовый артефакт и метрики
python3 ml/train.py --check  # проверить воспроизводимость
```

При изменении признаков поддерживайте совпадение Python и Go и обновляйте parity-тесты. Синтетические метрики описывают генератор данных; их нельзя выдавать за качество на пользователях. Протокол персонализации — в [ML-PRIORITIES.md](ML-PRIORITIES.md).

## Сборка релиза

`make backend-build` создаёт статический `linux/amd64` бинарник `build/rhytm` и его SHA-256. Его нельзя запускать напрямую на macOS; для локальной разработки используйте контейнер или `go run` с настроенными зависимостями.

Бинарник и checksum остаются в Git для существующего процесса доставки на VDS. `server`, `node_modules`, `.env`, логи и системные файлы в Git не входят. Изменение backend для production сопровождается пересборкой бинарника; изменения документации этого не требуют.
