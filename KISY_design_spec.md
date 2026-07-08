# KISY — дизайн-спецификация (Apple Pro dark)

Спецификация визуального стиля мессенджера KISY и экрана аудио-звонка. Цель — чтобы фронтенд-агент (Claude Code) сверстал реальные React + TypeScript компоненты точь-в-точь по этим значениям. Все величины — итоговые, не «примерно». Стек проекта не меняется: React + TS + Vite, FSD (`app/ · entities/ · features/ · pages/`).

Стиль: чистый чёрный фон, редкая точечная сетка, металлические и цветные градиентные заголовки, шрифт SF Pro. Тёмная тема — базовая.

---

## 1. Дизайн-токены

Заводи как CSS custom properties. Предлагаю файл `frontend/src/app/theme/tokens.css` (или в существующий global-стиль), подключаемый в `main.tsx`. Все компоненты используют только переменные, без хардкода hex.

### Поверхности (surfaces)

| Токен | Значение | Где |
|---|---|---|
| `--kisy-bg` | `#000000` | корневой фон приложения, лендинг |
| `--kisy-surface-1` | `#0a0a0c` | контейнер мессенджера, карточка звонка |
| `--kisy-surface-2` | `#141416` | выбранный чат, системные плашки |
| `--kisy-surface-3` | `#161618` | поле ввода, строка поиска |
| `--kisy-surface-chat` | `#050506` | фон ленты сообщений |
| `--kisy-bubble-in` | `#1c1c1f` | входящий пузырь, чужая карточка |

### Текст

| Токен | Значение | Где |
|---|---|---|
| `--kisy-text-primary` | `#f5f5f7` | имена, заголовки, текст сообщений |
| `--kisy-text-secondary` | `#8e8e93` | подписи, статусы, превью чата |
| `--kisy-text-muted` | `#6e6e73` | таймстемпы, плейсхолдеры |

### Границы (полупрозрачные — «наслаиваются» на чёрный)

| Токен | Значение | Где |
|---|---|---|
| `--kisy-border` | `rgba(255,255,255,0.08)` | базовый разделитель, шапки, панели |
| `--kisy-border-strong` | `rgba(255,255,255,0.09)` | внешняя рамка контейнеров и карточки звонка |
| `--kisy-border-subtle` | `rgba(255,255,255,0.06)` | поля ввода, строка поиска |
| `--kisy-border-faint` | `rgba(255,255,255,0.05)` | пузыри сообщений, системные плашки |

### Семантические цвета

| Токен | Значение | Где |
|---|---|---|
| `--kisy-accent` | `#2997ff` | ссылки, интерактивные акценты |
| `--kisy-accent-soft` | `#4aa3ea` | вторичный синий (индикаторы, волна) |
| `--kisy-success` | `#22b06b` | «в сети», принять звонок, ✓✓ |
| `--kisy-danger` | `#e2483d` | завершить/отклонить звонок |

---

## 2. Градиенты

Ключевой элемент стиля. Заводи как переменные и переиспользуй.

| Токен | Значение | Назначение |
|---|---|---|
| `--kisy-grad-brand` | `linear-gradient(145deg, #2b7fd4, #7a5ecc)` | аватары, брендовые круглые/квадратные кнопки, бейдж непрочитанного |
| `--kisy-grad-brand-h` | `linear-gradient(90deg, #2b7fd4, #7a5ecc)` | горизонтальный вариант: eyebrow, пилюля CTA |
| `--kisy-grad-silver` | `linear-gradient(180deg, #fbfbfd 0%, #c7c7cc 55%, #8e8e93 100%)` | крупные заголовки (метал) |
| `--kisy-grad-iris` | `linear-gradient(90deg, #4aa3ea, #9d7bff)` | подзаголовки-акценты, таймер звонка |
| `--kisy-grad-mint` | `linear-gradient(90deg, #22b06b, #4aa3ea)` | eyebrow секции «Аудио-звонки» |
| `--kisy-grad-accept` | `linear-gradient(145deg, #22b06b, #15c26a)` | кнопка «Принять» |
| `--kisy-grad-bubble` | `linear-gradient(135deg, #2b7fd4, #6a54c0)` | исходящий пузырь сообщения |
| `--kisy-grad-icon` | `linear-gradient(180deg, #ffffff, #8e8e93)` | крупные декоративные иконки (метал) |

### Градиентный текст — обязательный приём

Для металлических и цветных заголовков используй background-clip. Сделай утилиту-миксин/класс:

```css
.kisy-grad-text {
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
  /* background задаётся конкретным градиентом-токеном на элементе */
}
```

Применение (React):

```tsx
<h1 className="kisy-grad-text" style={{ backgroundImage: "var(--kisy-grad-silver)" }}>
  Разговор. Без лишнего.
</h1>
```

### Точечная сетка (pattern)

Редкий узор фона (герой лендинга, экран звонка):

```css
.kisy-dots {
  background-color: var(--kisy-bg);
  background-image: radial-gradient(rgba(255,255,255,0.045) 1px, transparent 1px);
  background-size: 20px 20px; /* 22–24px для крупных секций лендинга */
}
```

---

## 3. Типографика

- Семейство: `-apple-system, BlinkMacSystemFont, 'SF Pro Display', 'SF Pro Text', Inter, Helvetica, Arial, sans-serif`. Заведи `--kisy-font: …`.
- Начертания: **400** (обычный), **500** (полужирный акцент), **600** (только крупные заголовки/лендинг). Больше 600 не используем.
- Регистр — всегда обычный (sentence case), никаких CAPS.
- Таймеры и счётчики: `font-variant-numeric: tabular-nums`.

| Роль | Размер / вес | Цвет |
|---|---|---|
| Hero-заголовок (лендинг) | 72px / 600, letter-spacing −0.02em, line-height 1.02 | `--kisy-grad-silver` |
| Заголовок секции | 42px / 600, letter-spacing −0.015em | silver |
| Имя на экране звонка | 21px / 500 | text-primary |
| Таймер звонка | 22px / 500, tabular-nums | `--kisy-grad-iris` (grad-text) |
| Имя в списке чатов / шапке | 14px / 500 | text-primary |
| Текст сообщения | 13px / 400, line-height 1.5 | text-primary |
| Превью чата / статус | 12px / 400 | text-secondary |
| Таймстемп | 11px / 400 | text-muted |

---

## 4. Радиусы, отступы, эффекты

| Токен | Значение | Где |
|---|---|---|
| `--kisy-radius-container` | `18px` | контейнер мессенджера |
| `--kisy-radius-card` | `26px` | карточка звонка, «телефон» на лендинге |
| `--kisy-radius-btn` | `10px` | квадратные кнопки (звонок в шапке, edit) |
| `--kisy-radius-input` | `11px` | поле ввода |
| `--kisy-radius-search` | `9px` | строка поиска |
| `--kisy-radius-pill` | `980px` | CTA-пилюля |
| — | `50%` | все аватары и круглые кнопки звонка |

Пузыри сообщений: `border-radius: 14px 14px 14px 4px` (входящий), `14px 14px 4px 14px` (исходящий) — «хвостик» к своей стороне.

Границы всегда `1px solid` соответствующего токена. Никаких теней и blur внутри интерфейса (кроме `backdrop-filter: saturate(180%) blur(20px)` на липкой навигации лендинга). Скруглять только полные рамки — если используешь `border-left`-акцент (выбранный чат), радиус у этой стороны 0.

---

## 5. Компоненты

### 5.1 Экран аудио-звонка — `frontend/src/features/call/`

Общий контейнер (`CallScreen`): ширина 300px (модалка/оверлей), `--kisy-surface-1` + класс `kisy-dots`, рамка `--kisy-border-strong`, радиус `--kisy-radius-card`, padding `28px 22px`, flex-column, center.

Состояния (проп `status: 'incoming' | 'outgoing' | 'active'`):

**Верхняя строка** — flex space-between, 12px, text-secondary. Слева `ti-lock` + «зашифровано». Справа: `incoming`→«входящий», `active`→зелёный `ti-point-filled` + «соединено».

**Аватар** — 110×110, круг, фон `--kisy-grad-brand`, инициалы 40px/600 белым. margin `36px 0 18px`.

**Имя** — 21px/500. Под ним: `incoming/outgoing`→ роль (14px text-secondary) и «KISY звонит…» (13px text-muted, margin `22px 0 34px`); `active`→ таймер 22px grad-text (`--kisy-grad-iris`, tabular-nums) + эквалайзер.

**Эквалайзер (active)** — 6 полосок шириной 3px, высоты `[8,16,22,12,18,9]`px, gap 3px, border-radius 2px, цвета из iris-палитры (`#2b7fd4/#4aa3ea/#9d7bff/#7a5ecc`). Опционально анимировать высоту.

**Кнопки** — круглые, подпись 12px text-secondary под каждой, gap 9px.
- `incoming`: две кнопки 62×62 — «Отклонить» (`--kisy-danger`, `ti-phone-off`) и «Принять» (`--kisy-grad-accept`, `ti-phone`).
- `active`: три кнопки 54×54 — «Микрофон» (`ti-microphone`/`ti-microphone-off`, toggle mute; фон `--kisy-surface-2` + `--kisy-border`), «Динамик» (`ti-volume`, тот же стиль), «Завершить» (`--kisy-danger`, `ti-phone-off`).

Иконки в кнопках 22–26px, белые (у нейтральных — text-primary).

### 5.2 Список чатов — `pages/messenger/` (компонент `ChatList` / `ChatListItem`)

Панель 224px, справа `border-right: 1px var(--kisy-border)`.

Шапка панели: padding `12px 14px`, flex gap 10px. Лого 30×30, радиус 9px, `--kisy-grad-brand`, «K» 15px/600 белым. Название «KISY» 16px/500. Справа `ti-edit` 18px text-secondary.

Строка поиска: `--kisy-surface-3`, `--kisy-border-subtle`, радиус `--kisy-radius-search`, padding `7px 10px`, `ti-search` + «Поиск» 13px text-muted.

`ChatListItem`: flex gap 10px, padding `10px 14px`. Активный — фон `--kisy-surface-2` + `border-left: 2px solid #7a5ecc` (радиус слева 0). Аватар 42×42 круг: 1:1 — `--kisy-grad-brand`; группа — `linear-gradient(145deg,#7a5ecc,#9d7bff)` + `ti-users`; прочие — свои градиенты (см. макет: `#c76b3f→#e08a4a`, `#3f8f6b→#4fb383`). Онлайн-точка: 11×11 круг `--kisy-success`, снизу-справа, обводка 2px цветом фона строки.

Тело строки: верх — имя 14px/500 + таймстемп 11px text-muted (space-between); низ — превью 12px text-secondary (ellipsis, `white-space:nowrap; overflow:hidden`). Бейдж непрочитанного: `--kisy-grad-brand-h`, белый 11px, радиус 10px, padding `0 6px`. Пропущенный/входящий звонок в превью: `ti-phone`/`ti-phone-outgoing` 12px цветом `--kisy-success`.

### 5.3 Окно переписки — `pages/messenger/` (`ChatWindow`, `MessageBubble`, `MessageComposer`)

**Шапка чата**: padding `10px 16px`, `border-bottom: 1px var(--kisy-border)`, flex gap 12px. Аватар 36×36. Имя 14px/500 + статус «в сети» 12px `--kisy-success`. Затем **кнопка звонка** 36×36, радиус `--kisy-radius-btn`, фон `--kisy-grad-brand`, `ti-phone` 18px белым, `aria-label="Аудиозвонок"` — это главный акцент шапки, единственная залитая градиентом кнопка. Справа `ti-dots-vertical` 19px text-secondary.

**Лента**: фон `--kisy-surface-chat`, padding 16px, flex-column gap 10px, `justify-content:flex-end`.

`MessageBubble` (проп `direction`): max-width 70%, padding `8px 12px`, 13px.
- Входящий: `--kisy-bubble-in`, `--kisy-border-faint`, радиус `14px 14px 14px 4px`, слева.
- Исходящий: `--kisy-grad-bubble`, белый текст, радиус `14px 14px 4px 14px`, справа. Мета внизу 10px `#d6e4f5` (`float:right`, margin-left 8px): время + `✓✓`.

Системная плашка звонка (по центру): `--kisy-surface-2`, `--kisy-border-faint`, радиус 12px, padding `5px 12px`, 11px text-secondary, иконка `ti-phone-outgoing` 12px `--kisy-success`. Текст вида «Исходящий аудиозвонок · 4:17».

**Composer**: padding `10px 14px`, `border-top: 1px var(--kisy-border)`, flex gap 10px. `ti-paperclip` 20px text-secondary; поле `--kisy-surface-3` + `--kisy-border-subtle` радиус `--kisy-radius-input` padding `8px 12px`, плейсхолдер 13px text-muted; `ti-mood-smile` 20px; кнопка отправки/микрофона 36×36 круг `--kisy-grad-brand`, `ti-microphone` 18px белым (меняется на `ti-send` при наборе текста).

### 5.4 Лендинг — отдельная страница/маршрут

Готовая эталонная вёрстка лежит в `kisy_landing.html` — используй её как источник истины для секций (nav со `sticky` + `backdrop-filter`, hero с точечной сеткой, band, grid фич, cta, footer) и для scroll-reveal (IntersectionObserver, класс `.reveal → .in`, translateY 24px, opacity, transition 0.8s). Перенеси в React-компонент, вынеся стили в CSS-модуль/токены.

---

## 6. Иконки

Библиотека Tabler Icons (outline). Уже упоминается в стеке — подключи webfont или `@tabler/icons-react`. Только outline-начертания, без `-filled` (исключение: `ti-point-filled` для индикатора «соединено»). Используемый набор: `ti-lock, ti-phone, ti-phone-off, ti-phone-outgoing, ti-microphone, ti-microphone-off, ti-volume, ti-point-filled, ti-search, ti-edit, ti-users, ti-paperclip, ti-mood-smile, ti-send, ti-dots-vertical, ti-chevron-right, ti-bolt, ti-server`. Декоративным иконкам — `aria-hidden`, иконка-кнопкам — `aria-label`.

---

## 7. Definition of Done для вёрстки

- Реальные React + TS компоненты в `features/call/` и `pages/messenger/`, без инлайн-хардкода цветов — только токены из раздела 1–4.
- Значения совпадают с макетами: размеры аватаров, радиусы, отступы, градиенты — как указано выше.
- Экран звонка покрывает три состояния (`incoming/outgoing/active`) и toggle mute.
- Кнопка звонка в шапке чата залита `--kisy-grad-brand`, с `aria-label`.
- Доступность: контраст текста на чёрном проходит, все иконки-кнопки с `aria-label`, фокус-стили не удалены.
- Тёмная тема — базовая; если делаете светлую, выносите значения в отдельный набор токенов под `[data-theme="light"]`, компоненты не трогая.
- Собирается без ошибок `tsc`/Vite.
