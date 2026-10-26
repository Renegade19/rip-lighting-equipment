# LumiCalc — Лабораторная работа №4 (Go)

Тема: **Расчёт освещённости помещения светодиодными лампами**.
`Услуги` — модели светильников (мощность, угол рассеивания).

Веб-сервис для будущего SPA: отдаёт и принимает JSON, страниц не рисует.
В нём вся бизнес-логика приложения. Пользователь определяется по JWT,
вышедшие из системы токены хранятся в чёрном списке Redis, методы описаны в Swagger.

Стек: **Go + gin + PostgreSQL + gORM**, фото и видео — **Minio** (`minio-go`),
авторизация — **JWT** (`golang-jwt/jwt/v5`) и **Redis** (`go-redis/v9`),
документация — **Swagger** (`swaggo`), администрирование БД — **Adminer**
(хранилища поднимаются через `docker compose`).

Ветки: `lighting-equipment-lab1` — ЛР1 (коллекция в памяти, три GET),
`lighting-equipment-lab2` — ЛР2 (база данных, шесть HTTP-методов, HTML-страницы),
`lighting-equipment-lab3` — ЛР3 (веб-сервис, десять методов),
`lighting-equipment-lab4` — ЛР4 (JWT, Redis, Swagger).

## Структура проекта

```
cmd/lighting-equipment/main.go             # точка входа веб-сервиса, общее описание Swagger
cmd/migrate/main.go                        # создание таблиц (go run ./cmd/migrate)
internal/api/server.go                     # сборка сервиса: БД, Minio, Redis, маршруты
internal/app/
  config/config.go                         # настройки из .env: Minio, JWT, Redis
  dsn/dsn.go                               # строка подключения к PostgreSQL
  ds/                                      # модели: светильник, проектировщик, лайк
  ds/serializers.go                        # сериализаторы запросов и ответов (JSON)
  ds/jwt_claims.go                         # содержимое JWT: ид и логин проектировщика
  redis/redis.go                           # чёрный список JWT в Redis
  repository/repository.go                 # подключение к PostgreSQL (gORM) и Minio
  repository/lighting_equipment.go         # запросы домена "светильники" (ORM)
  repository/lighting_equipment_media.go   # загрузка и удаление файлов в Minio
  repository/lighting_user.go              # запросы домена "проектировщики" (ORM)
  handler/handler.go                       # маршруты, доступ гостя и создателя, Swagger UI
  handler/auth.go                          # middleware проверки JWT
  handler/lighting_equipment.go            # семь методов домена "светильники"
  handler/lighting_user.go                 # регистрация, вход (выдача JWT), выход
  handler/current_lighting_user.go         # текущий проектировщик из JWT
  handler/media.go                         # проверка файлов и генерация имён
assets/                                    # фото и видео для начального наполнения Minio
scripts/seed.sql                           # наполнение таблиц через Adminer
docs/docs.go, swagger.json, swagger.yaml   # документация Swagger (генерирует swag init)
docs/lumicalc.postman_collection.json      # коллекция запросов Postman
docs/control-questions.md                  # ответы на контрольные вопросы
docker-compose.yml                         # Minio, PostgreSQL, Adminer, Redis
```

## Таблицы БД

Три таблицы, каскадное удаление запрещено (все внешние ключи `ON DELETE RESTRICT`).

### `lighting_users` — проектировщики освещения (пользователи)

| Столбец | Тип | Ключ | Описание |
|---|---|---|---|
| `user_id` | serial | PK | идентификатор |
| `login` | varchar(50) | unique | логин |
| `full_name` | varchar(100) | | ФИО |
| `password_hash` | varchar(60) | | хэш пароля (bcrypt), клиенту не передаётся |

### `lighting_equipment` — светильники (услуги)

| Столбец | Тип | Ключ | Описание |
|---|---|---|---|
| `equipment_id` | serial | PK | идентификатор |
| `equipment_name` | varchar(150) | | наименование модели |
| `short_description` | varchar(500) | | краткое описание |
| `equipment_status` | enum | | черновик / опубликован / удалён |
| `image_url` | varchar(255) | | адрес файла изображения в Minio |
| `video_url` | varchar(255) | | адрес файла видео в Minio |
| `power_watt` | numeric(6,2) | | **поле по теме:** мощность, Вт |
| `beam_angle_deg` | integer | | **поле по теме:** угол рассеивания, ° |
| `created_at` | timestamp | | дата создания |
| `formed_at` | timestamp | | дата формирования (проставляется при публикации) |
| `creator_id` | integer | FK → `lighting_users` | создатель |

У проектировщика не более одного светильника в статусе «черновик» —
частичный уникальный индекс `uq_draft_per_creator`.

### `lighting_equipment_likes` — лайки, связь м-м

| Столбец | Тип | Ключ | Описание |
|---|---|---|---|
| `like_id` | serial | PK | идентификатор |
| `user_id` | integer | FK → `lighting_users` | кто поставил лайк |
| `equipment_id` | integer | FK → `lighting_equipment` | какому светильнику |

Пара `(user_id, equipment_id)` уникальна. Количество лайков нигде не хранится —
его считает БД подзапросом `COUNT(*)`.

## HTTP-методы

Все адреса начинаются с `/api`. Тела запросов и ответов — JSON,
кроме добавления светильника: там форма `multipart/form-data` с файлами.
Документация Swagger: http://localhost:3000/swagger/index.html

Доступ: **гость** — без токена; **создатель** — с заголовком
`Authorization: Bearer <accessToken>` из ответа метода 9.

| № | Домен | Метод | Адрес | Доступ | Назначение |
|---|---|---|---|---|---|
| 1 | светильники | GET | `/api/lighting-equipment` | гость | список опубликованных с фильтром по мощности |
| 2 | светильники | GET | `/api/lighting-equipment/feed`, `/api/lighting-equipment/:equipment_id` | гость | лента: один опубликованный светильник |
| 3 | светильники | GET | `/api/lighting-equipment/draft` | создатель | черновик текущего проектировщика |
| 4 | светильники | POST | `/api/lighting-equipment` | создатель | добавление светильника с фото и видео |
| 5 | светильники | PUT | `/api/lighting-equipment/:equipment_id/publish` | создатель | публикация: черновик → опубликован |
| 6 | светильники | DELETE | `/api/lighting-equipment/:equipment_id` | создатель | логическое удаление своего светильника |
| 7 | светильники | POST | `/api/lighting-equipment/:equipment_id/like` | создатель | поставить или отменить лайк |
| 8 | проектировщики | POST | `/api/lighting-users/register` | гость | регистрация |
| 9 | проектировщики | POST | `/api/lighting-users/login` | гость | аутентификация: выдаёт JWT |
| 10 | проектировщики | POST | `/api/lighting-users/logout` | создатель | выход: JWT попадает в чёрный список Redis |

Методы 1 и 2 работают с токеном и без него. С действующим токеном в ответе
заполняются признаки текущего проектировщика, без токена они равны 0.
Методам создателя без действующего токена сервер отвечает `401`.

### Домен «светильники»

**1. GET `/api/lighting-equipment?minPowerWatt=4.5&maxPowerWatt=10`** — список.
Оба параметра необязательны: без них отдаются все опубликованные.
Фильтрация выполняется в SQL. Ответ `200` — массив:

```json
[
  {
    "equipmentId": 1,
    "equipmentName": "SMART+ MATTER Filament Multicolor Classic Edison 40 4.8W E27",
    "imageUrl": "http://localhost:9000/lighting-equipment/vintage-1906-classic-a.jpg",
    "powerWatt": 4.8,
    "likesTotal": 4,
    "isCreatedByCurrentUser": 1
  }
]
```

`isCreatedByCurrentUser` — признак 0/1: создатель светильника совпадает с текущим проектировщиком.
Ошибки: `400` — параметр фильтра не число.

**2. GET `/api/lighting-equipment/feed`** — лента без ид: первый опубликованный светильник.
**GET `/api/lighting-equipment/3`** — светильник с указанным ид.
**GET `/api/lighting-equipment/3?next=true`** — следующий после указанного;
после последнего лента возвращается к первому. БД отдаёт сразу одну строку. Ответ `200`:

```json
{
  "equipmentId": 3,
  "equipmentName": "Vintage 1906 Big Oval 10 Filament Magnetic 4W 818 Smoke E27",
  "shortDescription": "Большая овальная филаментная лампа с дымчатой колбой.",
  "imageUrl": "http://localhost:9000/lighting-equipment/vintage-1906-big-oval-10-smoke-e27.jpg",
  "videoUrl": null,
  "powerWatt": 4,
  "beamAngleDeg": 320,
  "formedAt": "2026-02-03T11:20:00",
  "creator": { "userId": 2, "login": "i.svetov", "fullName": "Светов И. В." },
  "likesTotal": 3,
  "isLikedByCurrentUser": 0
}
```

`isLikedByCurrentUser` — признак 0/1: текущий проектировщик лайкнул этот светильник.
Ошибки: `400` — ид не число, `404` — светильника нет, он удалён или ещё не опубликован.

**3. GET `/api/lighting-equipment/draft`** — черновик текущего проектировщика, ид не указывается.
Ответ `200` — светильник целиком (как в методе 4). Ошибки: `404` — черновика нет.

**4. POST `/api/lighting-equipment`** — добавление. Форма `multipart/form-data`:

| Поле | Тип | Описание |
|---|---|---|
| `equipmentName` | текст | наименование, до 150 символов |
| `equipmentImage` | файл | изображение: jpeg, png, webp, gif, до 10 МБ |
| `equipmentVideo` | файл | короткое видео: mp4, webm, до 100 МБ |

Файлы сохраняются в Minio, в БД записываются их адреса. Ответ `201`:

```json
{
  "equipmentId": 7,
  "equipmentName": "LED Value Classic A 60 8.5W 827 E27",
  "shortDescription": null,
  "equipmentStatus": "черновик",
  "imageUrl": "http://localhost:9000/lighting-equipment/lighting-equipment-image-1791827182876-4cfd30ac.jpg",
  "videoUrl": "http://localhost:9000/lighting-equipment/lighting-equipment-video-1791827182876-d841ca36.mp4",
  "powerWatt": null,
  "beamAngleDeg": null,
  "createdAt": "2026-10-12T20:46:22",
  "formedAt": null,
  "creatorId": 1
}
```

Ошибки: `400` — нет названия или файла, неподдерживаемый тип файла;
`409` — у проектировщика уже есть черновик; `413` — запрос слишком большой.

**5. PUT `/api/lighting-equipment/7/publish`** — публикация. Тело:

```json
{ "shortDescription": "Светодиодная лампа для общего освещения.", "powerWatt": 8.5, "beamAngleDeg": 200 }
```

Все три поля обязательны. Статус меняется на «опубликован», дату формирования ставит сервер.
Ответ `200` — светильник целиком. Ошибки: `400` — поле не заполнено или вне диапазона;
`403` — светильник создан другим проектировщиком; `404` — светильника нет;
`409` — светильник уже опубликован.

**6. DELETE `/api/lighting-equipment/7`** — логическое удаление: запись остаётся в таблице,
статус меняется на «удалён». Ответ `204` без тела.
Ошибки: `403` — светильник создан другим проектировщиком; `404` — светильника нет или он уже удалён.

**7. POST `/api/lighting-equipment/3/like`** — лайк от текущего проектировщика. Тело:

```json
{ "equipmentLike": 1 }
```

`1` ставит лайк, `0` отменяет. Повторный запрос ничего не меняет. Ответ `200`:

```json
{ "equipmentId": 3, "likesTotal": 4, "isLikedByCurrentUser": 1 }
```

Ошибки: `400` — значение не 0 и не 1; `404` — светильника нет среди опубликованных.

### Домен «проектировщики»

**8. POST `/api/lighting-users/register`** — регистрация. Тело:

```json
{ "login": "n.fotonov", "password": "…", "fullName": "Фотонов Н. К." }
```

Логин: 3–50 символов, латиница, цифры, точка, дефис, подчёркивание. Пароль: от 6 символов.
В БД сохраняется хэш пароля. Ответ `201`:

```json
{ "userId": 5, "login": "n.fotonov", "fullName": "Фотонов Н. К." }
```

Ошибки: `400` — поле заполнено неверно; `409` — логин занят.

**9. POST `/api/lighting-users/login`** — аутентификация. Тело:

```json
{ "login": "a.parahnevich", "password": "lumicalc-demo" }
```

Сервер сверяет пароль с хэшем bcrypt и выдаёт JWT, подписанный HS256.
В токене — ид и логин проектировщика, срок жизни задаёт `JWT_EXPIRES_IN` (по умолчанию 1 час).
Ответ `200`:

```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9…",
  "tokenType": "Bearer",
  "expiresIn": 3600,
  "user": { "userId": 1, "login": "a.parahnevich", "fullName": "Парахневич А." }
}
```

Ошибки: `400` — не указан логин или пароль; `401` — неверный логин или пароль
(одна ошибка на оба случая, чтобы по ответу нельзя было узнать, есть ли такой логин).

**10. POST `/api/lighting-users/logout`** — выход, нужен заголовок `Authorization`.
Токен записывается в Redis с ключом `lumicalc.jwt.blacklist.<токен>`, значение — логин,
срок хранения — сколько токену оставалось жить. Ответ `200`:

```json
{ "status": "ok", "description": "проектировщик вышел из системы, токен больше не действует" }
```

Ошибки: `401` — нет токена, он недействителен или уже в чёрном списке.

### Формат ошибок

У всех методов ошибка приходит в одном виде:

```json
{ "status": "error", "description": "светильник не найден" }
```

## Правила бизнес-логики

- **Удалённые записи клиенту не передаются.** Список и лента работают только
  с опубликованными светильниками.
- **Статусы меняются только в одну сторону.** У создателя два метода:
  публикация (черновик → опубликован) и удаление (черновик или опубликован → удалён).
  Вернуть светильник в «черновик» нельзя, такого метода нет.
- **Системные поля с клиента не принимаются.** Ид, статус, создатель, дата создания
  и дата формирования вычисляются на сервере. В сериализаторах запросов этих полей нет,
  поэтому присланные значения игнорируются.
- **Менять светильник может только создатель.** Публикация и удаление чужого
  светильника отвечают `403`.
- **Текущий проектировщик берётся из JWT.** Middleware проверяет токен и кладёт его
  содержимое в контекст запроса, функция `getCurrentLightingUserID`
  (`internal/app/handler/current_lighting_user.go`) читает оттуда ид.
  В ЛР3 на этом месте была константа — вызовы в обработчиках не изменились.
  Создатель нового светильника заполняется автоматически из токена.
- **Вся работа с БД идёт через ORM** (gORM), включая логическое удаление,
  которое в ЛР2 выполнялось SQL-курсором.

## Авторизация

Middleware `WithAuthCheck` (методы создателя) и `WithOptionalAuth` (методы чтения)
в `internal/app/handler/auth.go` проверяют токен по шагам:

1. заголовок `Authorization` начинается с `Bearer `;
2. подпись HS256 совпадает с ключом `JWT_SECRET_KEY`, срок жизни не истёк;
3. токена нет в чёрном списке Redis.

Если хоть одна проверка не прошла, `WithAuthCheck` отвечает `401`,
а `WithOptionalAuth` выполняет запрос как от гостя.

Содержимое Redis можно посмотреть из командной строки:

```bash
docker exec -it lighting-equipment-go-redis redis-cli --scan --pattern 'lumicalc.jwt.blacklist.*'
docker exec -it lighting-equipment-go-redis redis-cli GET 'lumicalc.jwt.blacklist.<токен>'
docker exec -it lighting-equipment-go-redis redis-cli TTL 'lumicalc.jwt.blacklist.<токен>'
```

## Файлы в Minio

Фото и видео приходят в запросе добавления как файлы. Сервер проверяет размер
и тип по содержимому, а не по расширению, и сохраняет файлы в бакет `lighting-equipment`.
Имена генерируются на латинице и от исходного имени не зависят, например
`lighting-equipment-image-1791827182876-4cfd30ac.jpg`. Если запись в БД создать
не удалось, загруженные файлы из бакета удаляются.

## Запуск

1. Положить видео `smart-matter-classic-a.mp4` в папку `assets/`
   (в репозитории не хранится из-за размера; фото там уже лежат).

2. Поднять Minio, PostgreSQL, Adminer и Redis:
   ```bash
   docker compose up -d
   ```
   Бакет `lighting-equipment` создаётся автоматически, открывается на чтение,
   в него загружаются все файлы из `assets/`.
   Веб-консоль Minio: http://localhost:9001 (логин/пароль: `minioadmin` / `minioadmin`).
   Adminer: http://localhost:8081 (движок PostgreSQL, сервер `postgres`,
   пользователь `lighting`, пароль `lighting`, база `lighting_equipment`).

3. Скопировать `.env.example` в `.env`.

4. Создать таблицы:
   ```bash
   go run ./cmd/migrate
   ```

5. Наполнить таблицы через Adminer — выполнить содержимое `scripts/seed.sql`
   на вкладке «SQL-запрос».

6. Запустить веб-сервис:
   ```bash
   go run ./cmd/lighting-equipment
   ```
   Сервис отвечает по адресу http://localhost:3000/api,
   Swagger — http://localhost:3000/swagger/index.html.
   Учебные проектировщики из `seed.sql`: логины `a.parahnevich`, `i.svetov`,
   `m.lampova`, `d.luxov`, пароль у всех `lumicalc-demo`.

7. Импортировать в Postman коллекцию `docs/lumicalc.postman_collection.json`.
   Запрос «Аутентификация» сам сохраняет `accessToken` в переменную коллекции,
   методы создателя подставляют его в заголовок `Authorization`.
   В запросе добавления выбрать файлы фото и видео вручную, например из папки `assets/`.

8. После изменения аннотаций Swagger пересобрать документацию:
   ```bash
   go install github.com/swaggo/swag/cmd/swag@v1.16.6
   swag init -g cmd/lighting-equipment/main.go -o docs --parseInternal
   ```
