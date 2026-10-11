# Контрольные вопросы — Лабораторная работа №1

## 1. MVT (Model-View-Template)

MVT — вариант архитектурного паттерна MVC, часто используемый в связке
серверных фреймворков с шаблонизаторами (Django, а также любой backend
с server-side rendering, включая наш Go + gin + `html/template`):

- **Model** — представление данных предметной области. В проекте: структура
  `LightingEquipment` (`internal/app/ds/lighting_equipment.go`) и коллекция
  данных в `Repository` (`internal/app/repository`).
- **View** — обработчик, который выбирает данные и решает, какую страницу
  отдать (`internal/app/handler/lighting_equipment.go`).
- **Template** — HTML-шаблон с плейсхолдерами, в который подставляются
  данные модели (`templates/*.html`). Шаблоны обрабатывает стандартный пакет
  `html/template`, подключённый в `internal/api/server.go`
  (`r.LoadHTMLGlob("templates/*")`).

Отличие от классического MVC в названиях ролей: то, что в MVC называют
Controller, в MVT называется View, а то, что в MVC называют View, — Template.

## 2. Компоненты MVC в лабораторной работе

- **Model** — структура `LightingEquipment` и `Repository` (хранение и
  выборка данных: `GetPublishedLightingEquipment`,
  `GetLightingEquipmentForFeed`, `GetDraftLightingEquipment`).
- **View** — шаблоны `lighting-equipment-feed.html`,
  `lighting-equipment-draft.html`, `lighting-equipment-tile.html` и общая
  панель вкладок `lighting-equipment-tabbar.html`.
- **Controller** — `Handler` с тремя методами-обработчиками GET-запросов:
  они принимают HTTP-запрос, обращаются к Model (Repository) и передают
  данные в `ctx.HTML(...)` для рендеринга View.

Поток запроса: браузер → `Handler` (разбор URL и query) → `Repository`
(фильтрация, поиск, next) → `Handler` собирает данные для страницы →
`html/template` рендерит шаблон → HTML отдаётся клиенту.

## 3. Шаблонизация

Шаблонизация — подход, при котором HTML-страница собирается не вручную
конкатенацией строк, а через шаблон с плейсхолдерами и директивами
(`{{.EquipmentName}}`, `{{if}}`, `{{range}}`, `{{template}}`), которые на
сервере заполняются реальными данными перед отправкой клиенту.

Примеры в проекте:
- `{{range .LightingEquipment}}...{{else}}...{{end}}` в
  `lighting-equipment-tile.html` — рендер списка карточек, ветка `else`
  срабатывает на пустом списке.
- `{{if .Equipment.VideoURL}}...{{else}}...{{end}}` в
  `lighting-equipment-feed.html` — условный выбор video/img.
- `{{template "lighting-equipment-tabbar" .}}` — переиспользуемый блок
  (нижняя панель вкладок), подключаемый на всех трёх страницах.

Преимущества: разделение разметки и логики, переиспользование блоков,
более короткий и поддерживаемый код на сервере. Пакет `html/template`
экранирует данные с учётом контекста (текст, атрибут, URL), что защищает
от XSS.

## 4. HTTP и модель OSI

HTTP (HyperText Transfer Protocol) — прикладной протокол передачи
гипертекстовых данных, работающий поверх TCP (обычно порт 80/443).
В модели OSI (7 уровней) HTTP соответствует **7-му, прикладному уровню**
(Application Layer). Ниже него стоят:
- Transport (TCP) — надёжная доставка байтового потока;
- Network (IP) — маршрутизация пакетов;
- Data Link / Physical — передача кадров/сигналов по физической среде.

Каждый GET-запрос в нашем приложении (например `GET /lighting-equipment`) — это
HTTP-запрос: строка запроса (метод + URL + версия), заголовки (`Host`,
`Accept`, ...) и (для GET) пустое тело. Ответ сервера содержит статус-строку
(`200 OK` / `404 Not Found`), заголовки (`Content-Type: text/html`) и тело —
отрендеренный HTML.

## 5. Web

Web (Всемирная паутина) — система взаимосвязанных гипертекстовых документов,
доступных через интернет по протоколу HTTP(S), идентифицируемых через URI/URL
и отображаемых браузером. Основные составляющие Web: HTTP (протокол
передачи), HTML (язык разметки документов), URL/URI (адресация ресурсов),
браузер (клиент, интерпретирующий HTML/CSS/JS).

## 6. HTML

HTML (HyperText Markup Language) — язык разметки для описания структуры
веб-страниц с помощью тегов (элементов). Каждый элемент образует узел
DOM-дерева, которое браузер строит и отображает. В проекте HTML не пишется
руками для каждой страницы — он генерируется на сервере пакетом
`html/template` из файлов `templates/*.html` на основе данных модели
(см. пункт 3).
