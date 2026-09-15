# KISY Enterprise Technical Specification - Part 7 (Business Logic & User Flows)

## Registration Flow

CEO generates invitation → token valid 120 seconds → user submits token,
username and password → server validates token, creates account,
invalidates token, records audit event, opens first session.

## Login Flow

User authenticates, receives rotated session tokens, WebSocket
authenticates automatically, presence becomes online.

## Private Chat Flow

Higher level may initiate chat with lower level. Lower level cannot
initiate upward. Once chat exists both participants may exchange
messages until blocked.

## Group Flow

CEO creates group, assigns minimum level, users below that level never
see the group or its metadata.

## Message Lifecycle

Create → validate permissions → persist → publish through WebSocket →
update unread counters → audit if required → allow deletion according to
policy.

## File Upload

Scan, validate MIME, size, store, generate preview, attach to message,
notify recipients.

## Admin Flow

CEO manages users, roles, invitations, groups, backups, settings and
audit from dashboard.

## Лента сообществ: сортировка «популярные»

```
score = (reactors + 1) / (age_hours + 2) ^ 0.9
```

Ровно три коэффициента, и каждый отвечает за конкретную неприятность.

**`reactors` — число РАЗНЫХ людей, поставивших реакцию, а не число
реакций.** Одно мнение — один голос. С миграции 45 на пост и так ставится
не больше одной реакции от человека (`UNIQUE (post_id, user_id)`: другая
эмодзи заменяет первую, а не добавляется), но счёт всё равно идёт по
`DISTINCT user_id` — чтобы формула не зависела молча от этого ограничения.
В чатах правило прежнее: на сообщение можно поставить несколько разных
эмодзи — там реакция это быстрый ответ, а не голос.

**`+1` в числителе** — чтобы у свежего поста без реакций score не был
нулём. Иначе новый пост никогда не поднимется наверх, а значит и не
получит первую реакцию, чтобы подняться: замкнутый круг, в котором лента
показывает только старое.

**`+2` часа в знаменателе** — чтобы пост, опубликованный секунду назад, не
получал почти бесконечный score и не выбивал всё остальное на несколько
минут.

**Степень `0.9` — скорость остывания, и это главная ручка.** Она
подобрана под рабочий ритм: люди заходят в ленту раз в день, а не
обновляют её каждые десять минут, поэтому остывание медленнее, чем в
новостных агрегаторах (там 1.5–1.8). Что это значит на практике:

| Ситуация | Поведение |
|---|---|
| Хороший пост (≈10 человек отреагировали) | держится выше свежих пустых постов около суток |
| Тот же пост через двое суток | опускается ниже свежих |
| Хит недельной давности (50 реакций) | уже ниже любого свежего поста |

Крутить стоит именно степень: **больше** — лента живее и забывчивее,
**меньше** — дольше держит удачные посты. Числитель и смещение лучше не
трогать, они защищают от вырожденных случаев, а не настраивают вкус.

Score считается не на каждый запрос: он пересчитывается раз в 5 минут и
лежит в Redis. Лента с сортировкой «новые» (`GET /feed?sort=new`) score не
использует вовсе — это просто обратный хронологический порядок.

## Сообщества: кто что может

Сообщество говорит от своего имени. Пост в ленте и на стене подписан
названием и аватаром сообщества; кто из редакторов его опубликовал, читателям
не передаётся вовсе (в API нет поля автора) — это остаётся в
`posts.author_id` и журнале аудита для модерации.

| | Читатель (участник) | Редактор (владелец, редактор, модератор; CEO) |
|---|---|---|
| Стена, реакции | да | да |
| Публикация постов | нет | да |
| Доска и календарь | **нет** — ни вкладок, ни API (403) | да |
| Назначение карточки | — | только на того, кто сам видит доску |

**Добавить человека в сообщество нельзя никому** — ни владельцу, ни CEO
(403). В сообщество вступают сами: «Вступить» или заявка, которую одобряют
редакторы. Сообщество, способное записывать людей, показывало бы свои посты
тем, кто об этом не просил.

В обычной группе добавляет участников только её владелец или CEO. Раньше это
мог сделать любой, кто группу видит, — в том числе добавить себя в обход
вступления по заявке.

## Detailed Business Rule 1

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 2

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 3

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 4

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 5

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 6

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 7

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 8

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 9

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 10

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 11

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 12

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 13

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 14

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 15

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 16

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 17

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 18

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 19

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 20

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 21

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 22

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 23

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 24

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.

## Detailed Business Rule 25

Describe every validation, permission check, state transition, rollback,
audit event, notification, caching behavior, edge case, timeout, retry
policy, error response and recovery process for this user action.
