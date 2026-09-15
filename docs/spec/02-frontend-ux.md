# KISY Enterprise Technical Specification - Part 2 (Frontend & UX)

## Frontend Architecture

Use React + TypeScript + Vite with feature-based architecture.\
State management: Zustand.\
Server state: TanStack Query.\
Routing: React Router.\
Forms: React Hook Form + Zod validation.\
Internationalization prepared, default language Russian.\
WebSocket client isolated in service layer.

## Pages

Authentication, Registration (invitation token), Main Messenger,
Profile,\
Settings, Search, Group Management, User Management, Invitation
Management,\
Audit Log, Dashboard, System Settings, File Browser, Notification
Center.

## Messenger Layout

Three-column layout similar to Telegram Desktop.\
Left: chat list, search, pinned chats, favorites.\
Center: conversation.\
Right: context panel (members, media, files, links, permissions).\
Resizable panels, keyboard navigation, instant search.

## Apple-inspired Design

Dark theme only. Rounded corners (12-16 px), glassmorphism, subtle
gradients,\
high-quality typography, smooth 60 FPS animations, premium spacing,\
consistent iconography, elegant shadows, no visual clutter.

## UX Rules

Every action must have optimistic UI where safe.\
Loading skeletons instead of spinners.\
Infinite scrolling for messages.\
Lazy loading of media.\
Drag-and-drop uploads.\
Context menus.\
Accessible keyboard shortcuts.\
Unread counters update in real time.

## Components

Reusable Button, Modal, Dialog, Avatar, Badge, Toast, Dropdown,\
Tooltip, Tabs, SplitPane, ChatBubble, MessageInput, FileCard,\
EmojiPicker, ReactionBar, MemberList, AuditTable, DataGrid.

## Screen Specification 1

Describe every UI element, validation rule, loading state, empty state,
error state, permissions, transitions, accessibility behavior, keyboard
interaction, responsive behavior and API interaction for this screen.

## Screen Specification 2

Describe every UI element, validation rule, loading state, empty state,
error state, permissions, transitions, accessibility behavior, keyboard
interaction, responsive behavior and API interaction for this screen.

## Screen Specification 3

Describe every UI element, validation rule, loading state, empty state,
error state, permissions, transitions, accessibility behavior, keyboard
interaction, responsive behavior and API interaction for this screen.

## Screen Specification 4

Describe every UI element, validation rule, loading state, empty state,
error state, permissions, transitions, accessibility behavior, keyboard
interaction, responsive behavior and API interaction for this screen.

## Screen Specification 5

Describe every UI element, validation rule, loading state, empty state,
error state, permissions, transitions, accessibility behavior, keyboard
interaction, responsive behavior and API interaction for this screen.

## Screen Specification 6

Describe every UI element, validation rule, loading state, empty state,
error state, permissions, transitions, accessibility behavior, keyboard
interaction, responsive behavior and API interaction for this screen.

## Screen Specification 7

Describe every UI element, validation rule, loading state, empty state,
error state, permissions, transitions, accessibility behavior, keyboard
interaction, responsive behavior and API interaction for this screen.

## Screen Specification 8

Describe every UI element, validation rule, loading state, empty state,
error state, permissions, transitions, accessibility behavior, keyboard
interaction, responsive behavior and API interaction for this screen.

## Screen Specification 9

Describe every UI element, validation rule, loading state, empty state,
error state, permissions, transitions, accessibility behavior, keyboard
interaction, responsive behavior and API interaction for this screen.

## Screen Specification 10

Describe every UI element, validation rule, loading state, empty state,
error state, permissions, transitions, accessibility behavior, keyboard
interaction, responsive behavior and API interaction for this screen.

## Screen Specification 11

Describe every UI element, validation rule, loading state, empty state,
error state, permissions, transitions, accessibility behavior, keyboard
interaction, responsive behavior and API interaction for this screen.

## Screen Specification 12

Describe every UI element, validation rule, loading state, empty state,
error state, permissions, transitions, accessibility behavior, keyboard
interaction, responsive behavior and API interaction for this screen.

## Screen Specification 13

Describe every UI element, validation rule, loading state, empty state,
error state, permissions, transitions, accessibility behavior, keyboard
interaction, responsive behavior and API interaction for this screen.

## Screen Specification 14

Describe every UI element, validation rule, loading state, empty state,
error state, permissions, transitions, accessibility behavior, keyboard
interaction, responsive behavior and API interaction for this screen.

## Screen Specification 15

Describe every UI element, validation rule, loading state, empty state,
error state, permissions, transitions, accessibility behavior, keyboard
interaction, responsive behavior and API interaction for this screen.

## Emoji picker & reply jump (stage F)

The emoji picker (shared/ui/EmojiPicker) offers categorized emojis, keyword
search (RU/EN) and a "recent" row persisted in localStorage. It opens from
the composer (insert at the caret) and from a message's reaction menu (react
with any emoji, beyond the 5 quick ones). Clicking a message's reply badge
jumps to the original: it scrolls into view and flashes a highlight; if the
original is older than the loaded window, earlier pages are fetched until it
appears. Both are client-only — no API changes.

## Правило навигации: у каждого раздела ровно одна дверь

Меню, хаб и таббар успели обрасти копиями друг друга: Уведомления, Заметки,
Отзывы и Администрирование открывались из трёх мест, Профиль — из трёх.
Копии расходятся: пункт добавляют в одно место и забывают в двух других.
Поэтому распределение фиксированное.

**Таббар (телефон)** — экраны первого уровня и ничего больше:
Сообщения · Сообщества · [центральная кнопка: Хаб] · Рейтинг · Профиль.

Четвёртая позиция зависит от типа аккаунта, потому что у них разные
разделы, а не разные права на один и тот же:

- **приглашённый** (есть уровень) — там **Рейтинг**, а Лента живёт
  карточкой в Хабе;
- **обычный** (уровня нет) — рейтинга у него не существует, поэтому там
  **Лента**, и в Хабе её карточки уже нет.

Правило «одна дверь» при этом не нарушается: в каждый момент у каждого
раздела ровно одна дверь, просто набор разделов у двух типов разный.
Решает один хук `shared/lib/useCapabilities` — таббар, rail, Хаб и
guard'ы маршрутов спрашивают его, а не уровень напрямую. Уровень в
компонентах не сравнивают: `roleLevel` может отсутствовать, и голое
сравнение читает «нет уровня» как «сильнее CEO».

**Хаб** (центральная кнопка, на десктопе — пункт в rail) — всё
функциональное, что не является экраном первого уровня: Уведомления,
Голосования, Заметки, Отзывы, Звонки, Условия повышения, Администрирование
и быстрые действия.

**Drawer** (телефон, по аватару в шапке) — только то, чему больше негде
жить: карточка пользователя, переключатель темы, «Выйти». Навигации в нём
нет вовсе.

**Rail (десктоп)** — те же правила, что у таббара: Рейтинг, Чаты,
Сообщества, Хаб, Профиль, Выйти. Ничего из хаба здесь быть не должно.

Следствия, которые легко нарушить:

- Новый мелкий модуль подключается **только** карточкой в хабе. Если он
  появился ещё и в rail — это ошибка ревью.
- Раздел, доступный только из rail, на телефоне недостижим: rail там скрыт.
  Так уже терялись админка и история звонков.
- Переключатель темы живёт в одном компоненте (`features/profile/ThemeSwitcher`)
  и переиспользуется профилем и drawer — это один контрол в двух местах, а не
  две копии.
