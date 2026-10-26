# LumiCalc — Лабораторные работы №1–2 (Go)

Тема: **Расчёт освещённости помещения светодиодными лампами**.
`Услуги` — модели светильников (мощность, световой поток, угол рассеивания).
`Заявка` (в будущих лабораторных) — расчёт количества выбранных моделей светильников
для достижения требуемой освещённости в помещении заданной площади.

Стек: **Go + gin + html/template + PostgreSQL + gORM**,
фото и видео — **Minio**, администрирование БД — **Adminer**
(всё поднимается через `docker compose`).

Ветки: `lighting-equipment-lab1` — ЛР1 (коллекция в памяти, три GET),
`lighting-equipment-lab2` — ЛР2 (база данных, шесть HTTP-методов).

## Структура проекта

```
cmd/lighting-equipment/main.go             # точка входа приложения
cmd/migrate/main.go                        # создание таблиц (go run ./cmd/migrate)
internal/api/server.go                     # сборка сервера: БД, шаблоны, статика, маршруты
internal/app/
  config/config.go                         # настройки из .env, сборка url объектов Minio
  dsn/dsn.go                               # строка подключения к PostgreSQL
  ds/                                      # модели: светильник, проектировщик, лайк
  repository/repository.go                 # подключение к БД через gORM
  repository/lighting_equipment.go         # запросы: 5 через ORM, удаление — SQL UPDATE через курсор
  handler/handler.go                       # маршруты
  handler/lighting_equipment.go            # шесть обработчиков
  handler/current_lighting_user.go         # текущий проектировщик (константа)
  handler/media.go                         # фото и видео по умолчанию
templates/
  lighting-equipment-feed.html             # страница "Лента"
  lighting-equipment-draft.html            # страница "Добавление"
  lighting-equipment-tile.html             # страница "Плитка"
  lighting-equipment-tabbar.html           # нижняя панель вкладок
resources/
  css/lighting-equipment.css               # единый файл стилей (палитра и формы — из Figma)
  images/default-equipment.svg             # изображение по умолчанию
  videos/default-equipment.mp4             # видео по умолчанию
assets/                                    # фото и видео для загрузки в Minio
scripts/seed.sql                           # наполнение таблиц через Adminer
docker-compose.yml                         # Minio, PostgreSQL, Adminer
```

## Структура БД

Три таблицы, каскадное удаление запрещено (все внешние ключи `ON DELETE RESTRICT`).

### `lighting_users` — проектировщики освещения

| Столбец | Тип | Ключ | Описание |
|---|---|---|---|
| `user_id` | serial | PK | идентификатор |
| `login` | varchar(50) | unique | логин |
| `full_name` | varchar(100) | | ФИО |

### `lighting_equipment` — услуги (модели светильников)

| Столбец | Тип | Ключ | Описание |
|---|---|---|---|
| `equipment_id` | serial | PK | идентификатор |
| `equipment_name` | varchar(150) | | наименование модели |
| `short_description` | varchar(500) | | краткое описание |
| `equipment_status` | enum | | черновик / опубликован / удалён |
| `image_url` | varchar(255) | | url изображения в Minio |
| `video_url` | varchar(255) | | url видео в Minio |
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
оно считается подзапросом `COUNT(*)` на стороне БД.

## HTTP-методы (всего шесть)

| Метод | URL | Реализация | Назначение |
|---|---|---|---|
| GET | `/lighting-equipment?minPowerWatt=&maxPowerWatt=` | ORM | плитка: опубликованные + фильтр по диапазону мощности |
| GET | `/lighting-equipment/:equipment_id?next=true` | ORM | лента: БД возвращает одну строку |
| GET | `/lighting-equipment/draft` | ORM | черновик текущего проектировщика |
| POST | `/lighting-equipment/draft` | ORM | кнопка «Далее» — создание черновика |
| POST | `/lighting-equipment/publish` | ORM | кнопка «Опубликовать» — смена статуса + дата формирования |
| POST | `/lighting-equipment/delete` | **SQL `UPDATE` через курсор, без ORM** | логическое удаление с плитки |

С панели вкладок лента открывается без id — по адресу `/lighting-equipment/feed`
(тот же обработчик, показывает первый опубликованный светильник).

Фильтр на странице «Плитка» — два независимых ползунка (нижняя и верхняя границы
мощности) и кнопка «Применить». Форма отправляется обычным GET, фильтрация
выполняется в SQL (`power_watt BETWEEN ? AND ?`). Если нижняя граница уведена
выше верхней, обработчик меняет их местами.

Удаление: статус меняется SQL-запросом `UPDATE ... RETURNING equipment_id`,
который выполняется через курсор — `r.db.Raw(query, ...).Row()` возвращает
строковый указатель на результат, `row.Scan` читает id удалённой строки.
Запись остаётся в таблице.

Текущий проектировщик зафиксирован константой в
`internal/app/handler/current_lighting_user.go` (авторизация появится в ЛР4).

Файлы изображения и видео в этой лабораторной на сервер **не передаются**.
По кнопке «Далее» в таблицу записываются адреса объектов Minio, которых там нет.
Если поле url пустое либо файл по ссылке недоступен, подставляются
`resources/images/default-equipment.svg` и `resources/videos/default-equipment.mp4`,
лежащие вместе с иконками на SSR-сервере. В ленте у светильника без своего видео
показывается его фото; видео по умолчанию остаётся, когда нет ни видео, ни фото.

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

6. Запустить приложение:
   ```bash
   go run ./cmd/lighting-equipment
   ```
   Открыть http://localhost:3000 — редирект на страницу «Плитка».

## Что переносилось из референса

Цветовая палитра (акцентный оранжевый, тёмный фон ленты) и стиль карточек товаров
взяты из дизайна Figma по теме и с сайта каталога ламп LEDVANCE
(https://www.ledvance.com/en-int/professional-lighting/products/lamps-c6499) —
использовано только как стилистический референс (форма карточек, hover, нейминг моделей
в духе линейки "Vintage 1906"), без копирования текста/изображений сайта.
