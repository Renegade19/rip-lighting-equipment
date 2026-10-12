# LumiCalc — Лабораторная работа №3 (Go)

Тема: **Расчёт освещённости помещения светодиодными лампами**.
`Услуги` — модели светильников (мощность, угол рассеивания).

Веб-сервис для будущего SPA: отдаёт и принимает JSON, страниц не рисует.
В нём вся бизнес-логика приложения, кроме авторизации.

Стек: **Go + gin + PostgreSQL + gORM**, фото и видео — **Minio** (`minio-go`),
администрирование БД — **Adminer** (всё поднимается через `docker compose`).

Ветки: `lighting-equipment-lab1` — ЛР1 (коллекция в памяти, три GET),
`lighting-equipment-lab2` — ЛР2 (база данных, шесть HTTP-методов, HTML-страницы),
`lighting-equipment-lab3` — ЛР3 (веб-сервис, десять методов).

## Структура проекта

```
cmd/lighting-equipment/main.go             # точка входа веб-сервиса
cmd/migrate/main.go                        # создание таблиц (go run ./cmd/migrate)
internal/api/server.go                     # сборка сервиса: БД, Minio, маршруты
internal/app/
  config/config.go                         # настройки из .env, сборка адресов файлов Minio
  dsn/dsn.go                               # строка подключения к PostgreSQL
  ds/                                      # модели: светильник, проектировщик, лайк
  ds/serializers.go                        # сериализаторы запросов и ответов (JSON)
  repository/repository.go                 # подключение к PostgreSQL (gORM) и Minio
  repository/lighting_equipment.go         # запросы домена "светильники" (ORM)
  repository/lighting_equipment_media.go   # загрузка и удаление файлов в Minio
  repository/lighting_user.go              # запросы домена "проектировщики" (ORM)
  handler/handler.go                       # маршруты и единый формат ошибок
  handler/lighting_equipment.go            # семь методов домена "светильники"
  handler/lighting_user.go                 # три метода домена "проектировщики"
  handler/current_lighting_user.go         # функция-singleton: текущий проектировщик
  handler/media.go                         # проверка файлов и генерация имён
assets/                                    # фото и видео для начального наполнения Minio
scripts/seed.sql                           # наполнение таблиц через Adminer
docs/lumicalc.postman_collection.json      # коллекция запросов Postman
docs/control-questions.md                  # ответы на контрольные вопросы
docker-compose.yml                         # Minio, PostgreSQL, Adminer
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

### Домен «светильники»

| № | Метод | Адрес | Назначение |
|---|---|---|---|
| 1 | GET | `/api/lighting-equipment` | список опубликованных с фильтром по мощности |
| 2 | GET | `/api/lighting-equipment/feed`, `/api/lighting-equipment/:equipment_id` | лента: один опубликованный светильник |
| 3 | GET | `/api/lighting-equipment/draft` | черновик текущего проектировщика |
| 4 | POST | `/api/lighting-equipment` | добавление светильника с фото и видео |
| 5 | PUT | `/api/lighting-equipment/:equipment_id/publish` | публикация: черновик → опубликован |
| 6 | DELETE | `/api/lighting-equipment/:equipment_id` | логическое удаление своего светильника |
| 7 | POST | `/api/lighting-equipment/:equipment_id/like` | поставить или отменить лайк |

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

| № | Метод | Адрес | Назначение |
|---|---|---|---|
| 8 | POST | `/api/lighting-users/register` | регистрация |
| 9 | POST | `/api/lighting-users/login` | аутентификация — заглушка до ЛР4 |
| 10 | POST | `/api/lighting-users/logout` | деавторизация — заглушка до ЛР4 |

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

**9–10.** Методы аутентификации и деавторизации отвечают `501 Not Implemented`:
их логика появится в ЛР4.

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
- **Текущий проектировщик зафиксирован константой** в функции-singleton
  `getCurrentLightingUserID` (`internal/app/handler/current_lighting_user.go`).
  Авторизация появится в ЛР4 — изменится только эта функция.
- **Вся работа с БД идёт через ORM** (gORM), включая логическое удаление,
  которое в ЛР2 выполнялось SQL-курсором.

## Файлы в Minio

Фото и видео приходят в запросе добавления как файлы. Сервер проверяет размер
и тип по содержимому, а не по расширению, и сохраняет файлы в бакет `lighting-equipment`.
Имена генерируются на латинице и от исходного имени не зависят, например
`lighting-equipment-image-1791827182876-4cfd30ac.jpg`. Если запись в БД создать
не удалось, загруженные файлы из бакета удаляются.

## Запуск

1. Положить видео `smart-matter-classic-a.mp4` в папку `assets/`
   (в репозитории не хранится из-за размера; фото там уже лежат).

2. Поднять Minio, PostgreSQL и Adminer:
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
   Сервис отвечает по адресу http://localhost:3000/api.

7. Импортировать в Postman коллекцию `docs/lumicalc.postman_collection.json`.
   В запросе добавления выбрать файлы фото и видео вручную, например из папки `assets/`.
