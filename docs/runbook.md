# KISY — Runbook (эксплуатация и реагирование)

Оперативная шпаргалка: как поднять наблюдаемость, что означает каждый алерт и
что делать. Дополняется на каждом этапе hardening (O1–O7).

## Как запустить мониторинг

Стек наблюдаемости — оверлей поверх базового compose. Указывай файлы явно и
не забывай нужный оверлей окружения (dev-порты или prod-хардненинг):

```bash
# dev (публикует loopback-порты БД/бэкенда через override):
docker compose -f docker-compose.yml -f docker-compose.override.yml \
  -f docker-compose.monitoring.yml up -d

# prod (TLS + хардненинг контейнеров):
docker compose -f docker-compose.yml -f docker-compose.prod.yml \
  -f docker-compose.monitoring.yml up -d
```

> Как только передаёшь хоть один `-f`, автослияние `docker-compose.override.yml`
> отключается — поэтому в dev его надо указывать явно, иначе пропадут порты
> `18080`/`5432`/`6379`.

Поднимает: Prometheus (`:9090`), Alertmanager (`:9093`), Grafana (`:3000`) и
экспортёры `node`, `postgres`, `redis`, `blackbox` (внутренние, без публичных
портов). Все UI слушают только `127.0.0.1`.

### Telegram-алерты

1. Создай бота через **@BotFather** → получишь `TELEGRAM_BOT_TOKEN`.
2. Узнай `TELEGRAM_CHAT_ID` (например, через **@userinfobot**; для группового
   чата id отрицательный — не забудь добавить бота в чат).
3. Пропиши оба значения в `.env` (файл в `.gitignore`, в репозиторий не попадает):
   ```
   TELEGRAM_BOT_TOKEN=123456:AA...
   TELEGRAM_CHAT_ID=123456789
   ```
4. Перезапусти Alertmanager:
   `docker compose -f docker-compose.yml -f docker-compose.monitoring.yml up -d alertmanager`.

Конфиг Alertmanager рендерится из `deploy/monitoring/alertmanager.tmpl.yml`
подстановкой токена/chat_id из env при старте контейнера — секрет нигде не
коммитится.

### Проверка живости

- Правила и их состояние: `http://localhost:9090/alerts` и `/rules`.
- Таргеты скрейпа (все ли `UP`): `http://localhost:9090/targets`.
- Очередь алертов Alertmanager: `http://localhost:9093`.
- Тестовая отправка в Telegram (проверить связку Alertmanager→Telegram):
  ```bash
  docker compose -f docker-compose.yml -f docker-compose.monitoring.yml exec alertmanager \
    amtool alert add TestAlert severity=warning --alertmanager.url=http://localhost:9093 \
    --annotation=summary="runbook smoke test"
  ```
  Через несколько секунд в чат должно прийти сообщение.

## Каталог алертов и реакция

| Алерт | Severity | Значит | Что делать |
|---|---|---|---|
| **BackendDown** | critical | Prometheus не скрейпит backend >1м | `docker compose logs backend`; проверить `/ready`; перезапустить сервис; проверить БД/Redis |
| **PostgresDown** | critical | postgres-exporter не достучался до БД | Проверить контейнер `postgres`, диск (см. LowDisk), логи; при необходимости restore (ниже) |
| **RedisDown** | critical | redis-exporter не достучался до Redis | Проверить контейнер `redis`; presence/звонки/rate-limit деградируют, клиенты переподключатся |
| **ReadinessFailing** | critical | `/ready` отдаёт не-2xx >2м | Зависимость (БД/кэш) нездорова — смотреть Postgres/Redis; backend жив, но не готов обслуживать |
| **HighErrorRate** | critical | >5% ответов 5xx за 5м | Логи backend, недавний деплой; откатить при регрессе |
| **HighLatencyP95** | warning | p95 > 200мс за 10м | Насыщение пула БД (DBPoolSaturation), медленные запросы, нагрузка (см. O4) |
| **LowDisk** | critical | <15% свободного на `/` за 5м | Освободить место; ротация логов; **главная причина роста — вложения в БД (решается в O5)** |
| **PostgresConnectionsHigh** | warning | >80% от `max_connections` за 5м | Проверить утечки соединений, размер пула backend (O3), число реплик (O6) |
| **DBPoolSaturation** | warning | >80% пула pgx занято за 5м | Тюнинг `MaxConns` (O3); искать долгие/зависшие запросы |
| **NodeExporterDown** | warning | Нет метрик хоста >5м | Мониторинг диска/CPU ослеп — поднять node-exporter |
| **WorkerErrorsSpike** | warning | Воркер (`scheduled`/`disappear`/`attachments_cleanup`) >3 ошибок за 15м | Логи backend по префиксу воркера; проверить БД |

## Backup / restore (прод на Neon)

Прод-БД — внешний **Neon** (Postgres). Бэкап автоматизирован через **GitHub
Actions** (`.github/workflows/db-backup.yml`) — сервер держать не нужно:

- **ежедневно** (03:17 UTC) снимается дамп Neon (`pg_dump` клиентом v18 в
  контейнере), кладётся артефактом в GitHub (**30 дней**) и, если настроено,
  копируется в offsite-хранилище (S3/B2/R2);
- сразу же **restore-drill**: дамп восстанавливается во временный Postgres 18 и
  прогоняется `scripts/backup-smoke-check.sql` (миграции на месте и не dirty,
  ключевые таблицы читаются). Битый бэкап роняет workflow;
- **алерт**: упавший scheduled-workflow → GitHub шлёт письмо владельцу репо.
  (Хочешь в Telegram — можно добавить шаг с ботом, как в O1.)

### Секреты репозитория (Settings → Secrets and variables → Actions)

- **`DATABASE_URL`** (обяз.) — прямая строка Neon (`…?sslmode=require`).
- `BACKUP_GPG_PASSPHRASE` (опц.) — тогда дамп шифруется AES-256 «at rest».
- `S3_BUCKET` / `S3_ENDPOINT` / `S3_ACCESS_KEY_ID` / `S3_SECRET_ACCESS_KEY`
  (опц.) — выгрузка в S3-совместимое хранилище (Backblaze B2 / Cloudflare R2).

### Ручной бэкап по требованию

- Кнопкой: вкладка **Actions → DB backup → Run workflow**.
- Локально: `DATABASE_URL='…neon…' scripts/db-backup.sh backups`
  (нужен Docker; `BACKUP_GPG_PASSPHRASE` — чтобы зашифровать).

### Восстановление (recovery) в новую Neon-базу

1. Скачай свежий дамп: Actions → последний зелёный run → **Artifacts →
   db-backup**. (Или возьми локальный/offsite файл.)
2. Заведи новую базу на Neon, возьми её прямую строку.
3. Восстанови:
   ```bash
   TARGET_DATABASE_URL='postgresql://…new-neon…?sslmode=require' \
     scripts/db-restore.sh backups/kisy-YYYYMMDDTHHMMSSZ.sql.gz
   ```
   (`.gpg`-файл потребует `BACKUP_GPG_PASSPHRASE`.) Восстанавливай в **пустую**
   базу — скрипт грузит схему и данные, не дропая существующее.
4. В Render → сервис `kisy` → Environment → поменяй `DATABASE_URL` на новую
   строку → Deploy.

> `scripts/backup.sh` / `scripts/restore.sh` остаются для **локального
> docker-стека** (дампят контейнер `postgres`); для прода используй
> `db-backup.sh` / `db-restore.sh`, которые работают против внешнего URL.

## Пул соединений к БД (O3)

Пул pgx настраивается через env (дефолты в `.env.example`): `DB_POOL_MAX_CONNS`
(10), `DB_POOL_MIN_CONNS` (2), `DB_POOL_MAX_CONN_LIFETIME` (30m),
`DB_POOL_MAX_CONN_IDLE_TIME` (5m), `DB_POOL_HEALTHCHECK_PERIOD` (1m).

- **Главное правило:** `(число инстансов бэкенда) × DB_POOL_MAX_CONNS` должно
  оставаться заметно ниже `max_connections` сервера (Neon free ≈ 100), с
  запасом на админ-сессии и миграции. Это же ограничение действует при
  переходе на несколько реплик (O6).
- На serverless-Postgres (Neon) важны idle/lifetime: простаивающие соединения
  иначе держат compute «проснувшимся» или молча рвутся на стороне сервера.
- **Метрики** (уже отдаются с O1): `kisy_db_pool_acquired_conns`,
  `…_idle_conns`, `…_total_conns`, `…_max_conns`, `…_empty_acquire_count_total`.
  Алерт `DBPoolSaturation` срабатывает при >80% занятых соединений 5 минут.
- Признак нехватки пула: растёт `empty_acquire_count` и p95 латентности при
  спокойной БД → поднимай `DB_POOL_MAX_CONNS` (сверившись с лимитом сервера).
- Стартовые значения видны в логе: `connected to postgres pool_max_conns=… `.

## Файлы в объектном хранилище (O5)

Байты вложений и аватаров могут лежать не в Postgres, а в S3-совместимом
хранилище. Это снимает главную причину аутэйджа: БД перестаёт расти безгранично,
дампы становятся лёгкими, а переполнение диска БД больше не кладёт сайт.

**Включение** — задать в окружении бэкенда (см. `.env.example`):
`BLOB_S3_ENDPOINT`, `BLOB_S3_BUCKET`, `BLOB_S3_ACCESS_KEY`,
`BLOB_S3_SECRET_KEY` (+ `BLOB_S3_REGION`/`BLOB_S3_USE_SSL` для managed).
Пустой endpoint/bucket = прежнее хранение в БД. Self-hosted MinIO:
`docker compose -f docker-compose.yml -f docker-compose.storage.yml up -d`.
Бакет создаётся автоматически при старте.

**Как это устроено (важно для эксплуатации)**

- **Чтение всегда двухпутевое.** Строка с `storage_path` читается из
  хранилища, строка с инлайн-байтами — из БД. Поэтому включать хранилище можно
  до миграции: старые файлы продолжают работать.
- **Доступ не расширяется.** Файлы отдаёт бэкенд, публичные URL бакета клиенту
  не выдаются: права (членство в чате + клиренс) проверяются до чтения байтов,
  как и раньше. Presigned-ссылки намеренно не используются.
- **E2EE не затронуто** — в хранилище кладётся ровно то, что раньше лежало в
  БД (для E2EE-чатов это уже зашифрованный контент).
- **Проверка MIME и лимиты** работают как прежде — до записи в хранилище.
- **Исчезающие сообщения** удаляют и объект: каскад БД до бакета не достаёт,
  поэтому reaper собирает ключи и чистит их после коммита. Пересланная копия
  ссылается на тот же объект, поэтому удаляются только ключи, на которые не
  осталось ссылок.

**Перенос существующих файлов** (идемпотентно, возобновляемо, без простоя):

```bash
go run ./cmd/blobmigrate -dry-run       # посмотреть, что переедет
go run ./cmd/blobmigrate                # перенести БД -> хранилище
go run ./cmd/blobmigrate -direction=to-db   # вернуть обратно (перед откатом)
```

Каждая строка сначала пишется в хранилище и только потом освобождает колонку,
поэтому прерывание оставляет байты в обоих местах (безвредно) — следующий
проход дочистит. Откат миграции `000040` требует предварительного `-direction=to-db`.

**Бэкапы после переезда.** Дамп Postgres перестаёт содержать файлы, поэтому
бэкап бакета — отдельная задача: включи версионирование/lifecycle у провайдера
(B2/R2/S3) либо реплицируй бакет. Дамп БД по-прежнему делает workflow из O2.

## Нагрузочное тестирование (O4)

Сценарии k6 — `tests/load/` (см. README там же). Запуск: `make loadtest
CEO_PASSWORD='…'`, `make loadtest SCENARIO=ws VUS=100 …`. **Только против
локального/staging-стека, никогда против прода.**

### Что показали замеры (локально, 1 инстанс, 30s, 50 VU)

| Путь | Результат |
|---|---|
| REST через nginx (edge) | ~30 r/s на IP, остальное 429 — **упирается в rate-limit nginx**, не в приложение |
| REST напрямую в backend | **194 r/s, 0% ошибок, p95 = 11 мс** (бюджет 200 мс) |
| Отправка сообщения | p95 = 21 мс |
| WebSocket | сокеты держатся, доставка своего сообщения обратно p95 = 6 мс, handshake p95 = 7 мс |

Числа сняты на рабочей машине с соседними контейнерами — это порядок величины,
а не абсолютный потолок железа.

### Главные выводы

1. **Потолок «снаружи» задаёт nginx, а не Go.** `deploy/nginx/nginx.conf`:
   зона `api` — 30 r/s (burst 40) на IP, зона `auth` — 5 r/s. Это защита от
   одного злоупотребляющего клиента и работает как задумано; поэтому нагрузка
   с одного IP не измеряет ёмкость приложения — для этого бей в `backend:8080`
   внутри compose-сети (так и сделано выше).
2. **Пул БД — первое, что упирается внутри приложения.** Замер при 50 VU /
   ~190 r/s, одинаковая нагрузка, менялся только размер пула:

   | `DB_POOL_MAX_CONNS` | `empty_acquire` за прогон | p95 | RPS |
   |---|---|---|---|
   | 10 (дефолт) | 5198 | 14.7 мс | 193 |
   | 25 | 1902 (−63%) | 15.0 мс | 192 |

   То есть на этом уровне нагрузки дефолт ещё не бьёт по латентности, но
   очередь за соединениями видна. Порядок действий при росте нагрузки: следи
   за `kisy_db_pool_empty_acquire_count_total` — растёт → поднимай
   `DB_POOL_MAX_CONNS`, сверяясь с `max_connections` сервера и числом реплик.
3. **Auth-эндпоинты жёстко лимитированы** (register 5/мин, login 10/мин на IP),
   поэтому сидинг нагрузочных пользователей держим ≤ 5, а VU делят сессии.
   Повторные прогоны подряд надо разносить на минуту.

## Несколько инстансов и деплой без простоя (O6)

Прод-оверлей поднимает **2 инстанса бэкенда** (`deploy.replicas: 2` в
`docker-compose.prod.yml`). nginx резолвит имя сервиса `backend` **на каждый
запрос** (`resolver 127.0.0.11 valid=5s` + переменная в `proxy_pass`), поэтому
остановленный или заменённый контейнер перестаёт получать трафик без
перезагрузки nginx — обычный `upstream { server backend:8080; }` запомнил бы
адреса на старте и продолжил бить в мёртвый IP.

**Почему мультиинстанс безопасен** (проверено по коду):

| Состояние | Где живёт | Вывод |
|---|---|---|
| WebSocket-соединения | карта в памяти инстанса + мост Redis pub/sub | ок: сокет «липнет» к принявшему инстансу, события разносит Redis |
| Presence, состояние звонков, rate-limit | Redis | ок |
| Сессии чанковой загрузки | Postgres | ок |
| Воркеры (scheduled, disappear) | claim строк через `FOR UPDATE SKIP LOCKED` | ок: реплики не дублируют работу |
| Пакетные `map` в Go | только read-only справочники | ок |

Sticky-сессии не нужны: WS держится за свой инстанс сам, а REST не хранит
состояния между запросами.

**Масштабирование**

```bash
docker compose -f docker-compose.yml up -d --scale backend=2   # dev
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d  # prod (replicas: 2)
```

⚠️ Каждая реплика открывает **свой** пул к БД: `реплики × DB_POOL_MAX_CONNS`
должно с запасом влезать в `max_connections` сервера (см. раздел про пул).
При 2 репликах и дефолте 10 это 20 соединений.

**Rolling-деплой без простоя**

```bash
# 1. Собрать новый образ, не трогая работающие контейнеры
docker compose -f docker-compose.yml -f docker-compose.prod.yml build backend

# 2. Поднять на одну реплику больше — новая встаёт рядом со старыми
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d \
  --no-deps --scale backend=3 backend

# 3. Дождаться готовности новой (healthcheck бьёт в /health; убедись, что
#    /ready отвечает 200 — он проверяет Postgres и Redis)
docker compose -f docker-compose.yml -f docker-compose.prod.yml ps backend

# 4. Свернуть обратно до 2 — compose убирает самый старый контейнер
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d \
  --no-deps --scale backend=2 backend
```

Клиенты убитого инстанса теряют WebSocket и переподключаются автоматически
(фронтенд реконнектится), попадая на живую реплику; сообщения не теряются —
они уже в Postgres, а фан-аут идёт через Redis.

## Деплой / failover

Rolling-деплой нескольких инстансов — этап **O6**; HA данных и failover
Postgres/Redis — этап **O7** (разделы появятся на соответствующих этапах).
