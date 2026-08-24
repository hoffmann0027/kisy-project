# Android-приложение

KISY на телефоне — это не отдельный клиент, а тот же самый SPA-бандл в
нативной оболочке [Capacitor](https://capacitorjs.com). Оболочка живёт в
`frontend/android`, конфигурация — `frontend/capacitor.config.ts`.

Что даёт оболочка поверх сайта: иконка в списке приложений, push-уведомления
когда приложение закрыто, доступ к микрофону для звонков без запроса в
браузере, публикация в Google Play.

## Как это устроено

WebView отдаёт бандл со своего собственного origin — `https://localhost`, а
не с домена API. Отсюда два следствия, из-за которых нативная сборка ведёт
себя иначе, чем вкладка браузера:

- **Запросы кросс-доменные.** Cookie сессии (`SameSite=Strict`) не уходят,
  поэтому нативный клиент авторизуется Bearer-токеном
  (`frontend/src/shared/lib/native.ts`), а бэкенд пускает этот origin через
  `NATIVE_APP_ORIGINS`.
- **API-адрес абсолютный.** Он вшивается в бандл при сборке через
  `VITE_NATIVE_API_ORIGIN` (в CI — из переменной репозитория
  `NATIVE_API_ORIGIN`, по умолчанию `https://kisy.onrender.com`).

Схема `https` (а не `http`) выбрана намеренно: WebView считает такой origin
безопасным, а без этого не работают `getUserMedia` (звонки), WebCrypto
(ключи E2EE) и service worker.

## Сборка debug-APK (для своего телефона)

Локальный Android SDK не нужен — собирает CI:

1. GitHub → **Actions** → **Android build** → **Run workflow**.
2. Дождаться прогона, скачать артефакт **kisy-android-debug**.
3. Распаковать, перекинуть `.apk` на телефон, установить (Android спросит
   разрешение на установку из этого источника — это нормально для debug).

Прогон запускается и сам на каждый push в `main`, который трогает
`frontend/**`.

Если Android SDK и JDK 21 всё-таки стоят локально:

```bash
cd frontend && npm ci && npm run build && npx cap sync android && cd android && ./gradlew assembleDebug
```

## Push-уведомления (Firebase Cloud Messaging)

В браузере уведомления доставляет Web Push через service worker. В WebView
Push API нет, поэтому приложение получает их через FCM. Обе ветки скрыты за
одним фасадом `frontend/src/shared/lib/push.ts`, переключатель в профиле
одинаково работает и там, и там.

Что происходит на устройстве:

1. Пользователь включает уведомления в профиле → приложение спрашивает
   разрешение (Android 13+), создаёт канал `kisy_messages` с высокой
   важностью и получает от Firebase registration token.
2. Токен уходит на `POST /push/device` и хранится в таблице `device_tokens`.
3. Бэкенд шлёт уведомление через FCM HTTP v1 (`backend/internal/push/fcm.go`),
   параллельно с Web Push.
4. Тап по уведомлению открывает нужный чат — путь едет в `data.url`.

Токен перерегистрируется при каждом входе: FCM их ротирует, а протухший
токен перестаёт доставлять молча. При выходе из аккаунта устройство
отвязывается на сервере, но разрешение сохраняется — следующий вход
подхватит уведомления без повторного запроса.

Тексты уведомлений содержательно пустые («Новое сообщение») — содержимое
сообщений не покидает клиент.

### Что нужно настроить один раз

Без этих шагов приложение работает, но уведомления не приходят.

**1. Проект в Firebase**

1. [console.firebase.google.com](https://console.firebase.google.com) →
   **Add project** (Google Analytics можно отключить).
2. В проекте: **Add app → Android**. Package name — **`com.kisy.messenger`**
   (должен совпадать с `applicationId`, иначе push не дойдёт).
3. Скачать `google-services.json`.

**2. Секрет с конфигом приложения**

Файл содержит идентификаторы проекта, в репозиторий он не кладётся
(`frontend/android/.gitignore`). Кодируем в base64 и кладём в секреты.

Windows (PowerShell) — команда кладёт результат в буфер обмена:

```powershell
[Convert]::ToBase64String([IO.File]::ReadAllBytes("$HOME\Downloads\google-services.json")) | Set-Clipboard
```

Linux/macOS:

```bash
base64 -w0 google-services.json
```

GitHub → **Settings → Secrets and variables → Actions → New repository
secret**, имя **`GOOGLE_SERVICES_JSON`**, значение — вывод команды.

**3. Ключ сервисного аккаунта для бэкенда**

Firebase Console → **Project settings → Service accounts → Generate new
private key**. Скачается JSON — это полноценный ключ на отправку push от
имени всего проекта, обращаться с ним как с паролем БД.

Кодируем и кладём в переменную окружения бэкенда `FCM_SERVICE_ACCOUNT`
(на Render — Environment → Add secret; локально — в `.env`).

Windows (PowerShell):

```powershell
[Convert]::ToBase64String([IO.File]::ReadAllBytes("$HOME\Downloads\kisy-firebase-adminsdk.json")) | Set-Clipboard
```

Linux/macOS:

```bash
base64 -w0 kisy-firebase-adminsdk-xxxxx.json
```

Переменная принимает и сырой JSON, и base64. Пустая — мобильный push просто
выключен; битая — бэкенд не стартует, чтобы поломка не выглядела как
«уведомления иногда не приходят».

**4. Пересобрать APK** — с этого момента уведомления работают.

### Проверка

- `GET /api/v1/push/vapid-public-key` возвращает `mobileEnabled: true`.
- В логах бэкенда при старте: `mobile push enabled` с id проекта.
- В профиле включить уведомления, закрыть приложение (не свернуть, а
  закрыть), написать себе с другого аккаунта.

## Подписанный релиз для Google Play

Play принимает только `.aab`, подписанный **ключом загрузки** (upload key).
Ключ — это идентичность приложения в Play: потеряете его — обновлять
листинг будет нечем (восстановление возможно только через поддержку Google,
и только если включена Play App Signing). Держите копию вне GitHub.

**1. Создать ключ** (JDK, один раз):

```bash
keytool -genkey -v -keystore kisy-upload.jks -keyalg RSA -keysize 4096 -validity 10000 -alias kisy-upload
```

Пароль — длинный и случайный, в менеджер паролей. Файл `kisy-upload.jks`
никуда не коммитить (`*.jks` в `.gitignore`).

**2. Положить в секреты репозитория:**

| Секрет | Значение |
| --- | --- |
| `ANDROID_KEYSTORE_BASE64` | файл в base64 (PowerShell: `[Convert]::ToBase64String([IO.File]::ReadAllBytes("kisy-upload.jks")) \| Set-Clipboard`) |
| `ANDROID_KEYSTORE_PASSWORD` | пароль хранилища |
| `ANDROID_KEY_ALIAS` | `kisy-upload` |
| `ANDROID_KEY_PASSWORD` | пароль ключа (если отличается от пароля хранилища) |

**3. Собрать:** Actions → **Android release** → Run workflow → указать
версию (например `1.0.0`). На выходе артефакт с `.aab` (для Play) и `.apk`
(для прямой установки). Workflow проверяет, что артефакты действительно
подписаны, и удаляет keystore с раннера.

`versionCode` берётся из номера прогона — он всегда растёт, и Play не
отвергнет загрузку из-за повторного номера. `versionName` — то, что вы
ввели.

**4. Загрузить в Play Console:** Play Console → приложение → **Production**
(или Internal testing для обкатки) → **Create new release** → перетащить
`.aab`.

Что Play потребует помимо файла: иконка 512×512, скриншоты, описание,
политика конфиденциальности (URL), декларация о сборе данных (Data safety) —
приложение собирает сообщения, контакты внутри организации и аудио звонков.
Это разовая работа в консоли, кода не касается.

## Грабли и подводные камни

- **Package name менять нельзя.** `com.kisy.messenger` уже связан с
  Firebase-конфигом; в Play он вообще неизменяем после первой публикации.
- **`google-services.json` не в репозитории.** Сборка без него проходит, но
  push молча не работает — workflow предупреждает об этом в логе.
- **Веб-ассеты в `frontend/android/app/src/main/assets/public` генерируются**
  (`npx cap sync`) и в git не хранятся.
- **Файлы `capacitor.build.gradle` и `capacitor.settings.gradle` тоже
  генерируются**, но закоммичены — их перезаписывает `npx cap sync`. Правки
  туда вносить бессмысленно, менять нужно `variables.gradle` и
  `app/build.gradle`.
- **Права в `AndroidManifest.xml`** добавлены осознанно: интернет, микрофон и
  `MODIFY_AUDIO_SETTINGS` для звонков, `WAKE_LOCK` для звонка при погашенном
  экране, `POST_NOTIFICATIONS` для Android 13+.

## Смежные документы

- [mobile.md](mobile.md) — адаптив и правила вёрстки под телефон.
- [security.md](security.md) — модель угроз, в т.ч. Bearer-авторизация
  нативного клиента.
- [deploy-render.md](deploy-render.md) — прод, на который смотрит приложение.
