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

## Деплой / failover

Rolling-деплой нескольких инстансов — этап **O6**; HA данных и failover
Postgres/Redis — этап **O7** (разделы появятся на соответствующих этапах).
