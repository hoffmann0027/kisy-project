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

## Backup / restore

Резервное копирование и восстановление БД — `scripts/backup.sh` /
`scripts/restore.sh`. Автоматизация расписания, offsite и restore-drill —
этап **O2** (этот раздел будет расширен).

## Деплой / failover

Rolling-деплой нескольких инстансов — этап **O6**; HA данных и failover
Postgres/Redis — этап **O7** (разделы появятся на соответствующих этапах).
