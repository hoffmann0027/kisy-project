repo: hoffmann0027/kisy-project
branch: main
path: frontend/src

## Last sync
date: 2026-08-26T10:25:35Z

### Updated in this project
- Собран редизайн мобильного KISY: таб-бар с орбом, Хаб, drawer, Сообщения, Чат
- Кнопки, поля, бейджи и аватары сверены с `shared/ui/ui.css`; расхождения помечены в галерее
- Иконки взяты из `shared/ui/icons.tsx` (не Tabler, как в спеке) — изменён только штрих
- Рейл получил safe-area математику из `theme.css`; `--brand-grad` помечен как новый токен

## Screen map
| Экран в проекте | Файлы в репозитории |
|---|---|
| KisyPhone — токены, цвета, радиусы, отступы | frontend/src/shared/config/theme.css |
| KisyPhone — иконки | frontend/src/shared/ui/icons.tsx |
| KisyPhone — кнопки, поля, бейдж, аватар | frontend/src/shared/ui/ui.css, frontend/src/shared/ui/{Button,Input,Badge,Avatar,IconButton}.tsx |
| KisyPhone — темы (glass/luce/aurora/cyber/xp/matrix) | frontend/src/shared/store/theme.ts |
| KISY Redesign — токены, UI-кит, иконки | frontend/src/shared/config/theme.css, frontend/src/shared/ui/ui.css, frontend/src/shared/ui/icons.tsx |
| Не прочитано (заглушки: Сообщества, Рейтинг, Профиль) | frontend/src/pages/messenger/, frontend/src/pages/ |

## Notes
- `KISY_design_spec.md` описывает другую палитру (Apple Pro dark, `#2997ff`, SF Pro, Tabler Icons) для лендинга и экрана звонка. Мобильный редизайн ведётся по брифу и текущему UI, не по этой спеке.
