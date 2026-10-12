# Домашнє завдання: REST API на Go

Ви реалізуєте REST API для **власної сутності** (ваш варіант). Усі варіанти мають **однаковий контракт**,
тому правильність перевіряється автоматичними тестами. Ви пишете сервіс самі (з ШІ-помічником чи без),
але маєте розуміти кожен рядок свого коду: це перевіряється на захисті.

## 1. Як почати

1. Створіть свій репозиторій із цього шаблону (кнопка **Use this template** або посилання від викладача).
2. Відкрийте файл `VARIANT` і запишіть назву **вашого варіанта** (див. таблицю нижче), наприклад `movies`.
3. Запустіть `go test ./...`. Усі тести спочатку **червоні**, це нормально: ви рухаєтеся до зеленого.
  На macOS з Go 1.22 використовуйте `sh scripts/test.sh ./...`: скрипт вмикає зовнішній linker, щоб обійти відсутній `LC_UUID` у тестових бінарниках.
4. Реалізуйте `NewRouter()` у `internal/app/app.go` й усе, що потрібно для нього.
5. Після кожного `git push` GitHub Actions запускає перевірку й показує бали на сторінці запуску (розділ *Summary*).

> Тести звертаються до вашого API **лише** через `app.NewRouter() http.Handler`. Структуру решти коду, роутер
> (`net/http`, `gin`, `chi`...) і бібліотеки обираєте ви. Файли `*_test.go`, `scripts/` і `.github/workflows/ci.yml`
> **не змінюйте**: під час оцінювання використовуються оригінальні версії.

## 2. Варіанти

Кожна сутність має **4 обов'язкові рядкові поля**, **одне необов'язкове ціле** і **одне необов'язкове текстове** поле.
Четверте обов'язкове поле є полем **фільтра**.

| Варіант (`VARIANT`) | Ресурс у URL | Обов'язкові поля (4-те це фільтр) | Необов'язкове ціле | Необов'язковий текст |
|---|---|---|---|---|
| `books` | `books` | `title`, `isbn`, `author`, `category` | `published_year` | `description` |
| `movies` | `movies` | `title`, `director`, `country`, `genre` | `release_year` | `synopsis` |
| `tasks` | `tasks` | `title`, `assignee`, `project`, `status` | `priority` | `notes` |
| `recipes` | `recipes` | `name`, `author`, `difficulty`, `cuisine` | `cook_minutes` | `instructions` |
| `devices` | `devices` | `name`, `serial`, `manufacturer`, `type` | `warranty_months` | `comment` |
| `courses` | `courses` | `title`, `teacher`, `level`, `subject` | `hours` | `program` |
| `albums` | `albums` | `title`, `artist`, `label`, `genre` | `release_year` | `notes` |
| `pets` | `pets` | `name`, `owner`, `breed`, `species` | `age` | `notes` |

Нижче в прикладах використано варіант `books`. Підставте своє.

## 3. Контракт API

Усі відповіді (крім `204`) мають `Content-Type: application/json`. Ключі JSON пишуться в `snake_case`.

| Метод і шлях | Успіх | Помилки |
|---|---|---|
| `GET /health` | `200` `{"status":"ok"}` | |
| `POST /api/v1/books` | `201` + створений ресурс | `422` |
| `GET /api/v1/books` | `200` + масив | `400` (`invalid_pagination`) |
| `GET /api/v1/books/{id}` | `200` + ресурс | `400` (`invalid_id`), `404` |
| `PUT /api/v1/books/{id}` | `200` + збережений ресурс | `400`, `422`, `404` |
| `DELETE /api/v1/books/{id}` | `204` без тіла | `400`, `404` |

Невідомий маршрут дає `404`, непідтримуваний метод (наприклад `PATCH`) дає будь-який `4xx`.

### Ресурс

```json
{
  "id": 1,
  "title": "The Go Programming Language",
  "isbn": "9780134190440",
  "author": "Donovan, Kernighan",
  "category": "programming",
  "published_year": 2015,
  "description": null,
  "created_at": "2026-10-05T12:00:00.123456Z",
  "updated_at": "2026-10-05T12:00:00.123456Z"
}
```

- `id`: додатне ціле, починається з `1`, зростає на 1, **ніколи не використовується повторно** після видалення.
- Необов'язкові поля **завжди присутні** у відповіді, `null`, якщо не були задані.
- `created_at` і `updated_at`: час у форматі RFC 3339. Створення виставляє обидва. `PUT` змінює лише `updated_at`.

### Валідація (`422`, код `validation_error`)

- Обов'язкове поле відсутнє, `null`, не рядок, порожнє або складається лише з пробілів.
- Необов'язкове поле має хибний тип (рядок замість числа, дробове число замість цілого, число замість тексту).
- Тіло запиту не є коректним JSON-об'єктом (зіпсований JSON, порожнє тіло, масив).

### Ідентифікатор

`{id}` має бути цілим числом від `0` до `4294967295` (`uint32`). Інакше (`abc`, `-1`, `1.5`, завелике число)
це `400` з кодом `invalid_id`. Коректний, але неіснуючий `id` це `404` з кодом `not_found`.
Для `PUT` порядок перевірок такий: **спочатку `id` (400), потім тіло (422), потім існування (404)**.

### `PUT`: повна заміна

Тіло такого ж формату, як при створенні. Обов'язкові поля замінюються, **необов'язкові, яких немає в тілі, стають
`null`**. Відповідь містить збережений ресурс (з тим самим `id` і `created_at`). Запит, відхилений валідацією,
нічого не змінює.

### Список

`GET /api/v1/books?page=1&limit=10&category=programming`

- Порожня колекція повертається як `[]`, а не `null`.
- Порядок: за `id` за зростанням, **завжди однаковий**.
- `page` (за замовчуванням `1`) і `limit` (за замовчуванням `10`) мають бути цілими числами `>= 1`, інакше
  `400` з кодом `invalid_pagination`. Якщо `limit > 100`, використовується `100`. Сторінка за межами даних дає `[]`.
- Фільтр: параметр запиту має **ту саму назву, що й четверте обов'язкове поле** (`category`, `genre`, `status`...).
  Збіг точний і чутливий до регістру. Порожнє значення (`?category=`) означає «без фільтра».
  Фільтр працює разом із пагінацією.
- Дуже великі значення (`page=9223372036854775807`) не повинні призводити до паніки чи `500`.

### Формат помилки

```json
{ "error": { "code": "not_found", "message": "Book not found" } }
```

Коди: `invalid_id`, `invalid_pagination`, `validation_error`, `not_found`, `internal_server_error`.
`message` це непорожній рядок. Відповідь **не повинна розкривати внутрішні деталі** (текст паніки, stack trace,
імена файлів). Паніка в обробнику перетворюється на `500` з кодом `internal_server_error`, а подробиці йдуть у лог.

### CORS, конкурентність, стан

- Кожна відповідь має заголовок `Access-Control-Allow-Origin`. Запит `OPTIONS` дає `204` із
  `Access-Control-Allow-Methods` (зі списком методів, зокрема `POST`).
- Сервіс безпечний для паралельних запитів (перевіряється з `go test -race`).
- Кожен виклик `NewRouter()` створює **нове порожнє сховище** (без глобальних змінних зі станом).

## 4. Етапи (рекомендований порядок)

| Етап | Тести | Що робимо |
|---|---|---|
| 1 | `TestStage1_*` | каркас, `/health`, створення й отримання одного ресурсу |
| 2 | `TestStage2_*` | валідація й помилки (`400`, `404`, `422`) |
| 3 | `TestStage3_*` | `PUT`, `DELETE`, ідентифікатори, час |
| 4 | `TestStage4_*` | список, сортування, пагінація, фільтр |
| 5 | `TestStage5_*` | CORS і конкурентність |

Запуск одного етапу: `go test ./internal/app -run TestStage2 -v`. Повна перевірка: `go test -race ./...`.
Окрім видимих тестів, є **приховані** (граничні випадки з цього опису), які запускає викладач.

## 5. Рекомендована структура

```
cmd/api/main.go              запуск сервера (готовий)
internal/app/app.go          NewRouter(): збирає все докупи
internal/model/              структури ресурсу і запитів
internal/repository/         інтерфейс сховища + реалізація в пам'яті (з мʼютексом!)
internal/handler/            HTTP-обробники
internal/middleware/         логування, відновлення після паніки, CORS
```

Типові помилки, на які варто звернути увагу (їх регулярно допускає згенерований код): випадковий порядок
елементів при обході `map`, повернення вказівників на внутрішні об'єкти сховища (гонка даних), `panic` при від'ємних
межах зрізу в пагінації, відсутній час створення, відповідь `PUT` без `id`, розкриття тексту паніки клієнту.

## 6. Оцінювання (100 балів)

| Частина | Бали |
|---|---|
| Видимі автотести (етапи 1-5) | 40 |
| Приховані автотести | 20 |
| `gofmt` і `go vet` без зауважень | 5 |
| Якість коду: шари, інтерфейс сховища, README, Swagger, історія комітів | 15 |
| Захист (пояснити код, змінити вимогу наживо) | 20 |

Автоматична перевірка у вашому репозиторії показує видимі тести та `gofmt`/`vet` (до 45 балів).

## 7. Здача

Репозиторій із коректним `VARIANT`, зеленим CI і короткою секцією в цьому README: як запустити, що реалізовано,
які рішення ви прийняли. Дедлайн і форма захисту: за вказівкою викладача.

## Запуск по HTTP

Виконуйте:

```bash
go run ./cmd/api
```

Сервер працює на `http://localhost:8080`. Для перевірки статусу запустіть:

```bash
curl http://localhost:8080/health
```

Відповідь:

```json
{"status":"ok"}
```

Порт можна змінити через змінну середовища `PORT`:

```bash
PORT=8081 go run ./cmd/api
```

Варіант читається з `VARIANT`; у цьому репозиторії встановлено `books`. HTTPS використовується лише за потреби і потребує вказати `TLS_CERT_FILE` та `TLS_KEY_FILE`.

Реалізовано:

- повністю, ізоловане сховище для кожного `NewRouter()`;
- типізовані CRUD-операції, валідацію, пагінацію, фильтр і сортування;
- RFC 3339 timestamps, уникальні ID та заміну всього ресурсу через `PUT`;
- JSON помилки, CORS, OPTIONS та захист від панікі;
- конкурентну роботу через mutex + RWMutex;
- запуск через `cmd/api/main.go`.

Проверка:

```bash
go test -race ./...
go vet ./...
gofmt -l .
```

## Lesson 8 Report

### Run locally

```sh
cp .env.example .env
docker compose up -d postgres
set -a
. ./.env
set +a
go run ./cmd/api
```

`.env.example` is the committed template; `.env` is the local configuration read automatically by Docker Compose and ignored by Git. The shell export makes the same variables available to `go run`. The application runs migrations before accepting requests. Migration `000001_init_schema` represents the previous text-field schema. Migration `000002_normalize_books` moves categories and authors into separate tables, creates the many-to-many join table and foreign keys, and adds foreign-key indexes. Migration `000003_seed_demo_data` adds 10 books, 5 authors, 3 categories, copies, and two users. Each migration has a matching down file.

The production request path is `cmd/api -> app.NewRouterWithDB -> handler.LibraryAPI -> service.Library -> repository interfaces -> GORM`. HTTP handlers contain request parsing and response mapping; reference validation and book/copy use cases live in the service; each GORM query is encapsulated by a repository. The original in-memory `app.NewRouter()` remains for the Lesson 7 stage tests, while the running server uses `NewRouterWithDB`.

### Migration prompt

> Generate `000002_normalize_books.up.sql` and `000002_normalize_books.down.sql` for golang-migrate PostgreSQL migrations. The starting schema has a `books` table with `id`, `title`, `isbn`, `author` (text, possibly a comma-separated list), and `category` (text). Normalize it to `authors(id, full_name, bio)`, `categories(id, name)`, `books` with `category_id`, and the many-to-many join table `book_authors(book_id, author_id)`. Preserve existing book data: create distinct author/category rows, split comma-separated author names, and populate both relationships before dropping the old text columns. Add explicit named foreign-key constraints and indexes on foreign-key columns. Use `ON DELETE RESTRICT` for a book's category and `ON DELETE CASCADE` for rows in `book_authors` when a book is deleted. Make the down migration reverse the transformation: restore the text columns and values first, then drop dependent tables, constraints, and indexes in reverse dependency order. Also create `copies` and `users` tables with their foreign keys/check constraints for this application. Do not use ORM auto-migration.

The down migration was exercised against a legacy book containing two comma-separated authors; rolling back restored its original `author` and `category` text values.

### N+1 evidence: BEFORE FIX

The baseline was reproduced locally by loading each book's authors and category inside a loop. The sample database contained 10 books. One list request executed 21 SELECT statements: one books query, then one author query and one category query per book. Representative log lines:

```text
[8.772ms] [rows:10] SELECT * FROM "books" ORDER BY id ASC LIMIT 100
[2.346ms] [rows:2] SELECT "authors"."id","authors"."full_name","authors"."bio","authors"."created_at","authors"."updated_at" FROM "authors" JOIN "book_authors" ON "book_authors"."author_id" = "authors"."id" AND "book_authors"."book_id" = 1
[0.922ms] [rows:1] SELECT * FROM "categories" WHERE "categories"."id" = 1 ORDER BY "categories"."id" LIMIT 1
[0.357ms] [rows:2] SELECT "authors"."id","authors"."full_name","authors"."bio","authors"."created_at","authors"."updated_at" FROM "authors" JOIN "book_authors" ON "book_authors"."author_id" = "authors"."id" AND "book_authors"."book_id" = 2
[0.339ms] [rows:1] SELECT * FROM "categories" WHERE "categories"."id" = 1 ORDER BY "categories"."id" LIMIT 1
...
[0.289ms] [rows:2] SELECT "authors"."id","authors"."full_name","authors"."bio","authors"."created_at","authors"."updated_at" FROM "authors" JOIN "book_authors" ON "book_authors"."author_id" = "authors"."id" AND "book_authors"."book_id" = 10
[0.283ms] [rows:1] SELECT * FROM "categories" WHERE "categories"."id" = 3 ORDER BY "categories"."id" LIMIT 1
```

### N+1 evidence: AFTER FIX

The final `List` uses `.Preload("Authors").Preload("Category")` and joins `Category` into the books query. GORM recognizes the already joined category and does not issue a redundant preload query. It consistently uses three SELECT statements, independent of the number of books. Captured GORM log excerpt:

```text
[3.283ms] [rows:18] SELECT * FROM "book_authors" WHERE "book_authors"."book_id" IN (1,2,3,4,5,6,7,8,9,10)
[1.528ms] [rows:5] SELECT * FROM "authors" WHERE "authors"."id" IN (1,2,3,4,5)
[23.193ms] [rows:10] SELECT "books"."id","books"."title","books"."isbn","books"."category_id","books"."created_at","books"."updated_at","Category"."id" AS "Category__id","Category"."name" AS "Category__name","Category"."created_at" AS "Category__created_at","Category"."updated_at" AS "Category__updated_at" FROM "books" LEFT JOIN "categories" "Category" ON "books"."category_id" = "Category"."id" ORDER BY id ASC LIMIT 100
2026/10/12 13:41:57 INFO HTTP request method=GET path=/api/v1/books status=200 duration=15ms
```

GORM's many-to-many association needs one batch query for the join table and one for authors; joining the single-valued category into the main query keeps the total at three. The response was checked to include each book's category and authors.

### Latency

Measured locally with five sequential `curl` requests to `GET /api/v1/books?limit=100`, against the same isolated PostgreSQL database containing the same 10 seeded books. One unmeasured warm-up request was run before each five-request sample.

| Implementation | Samples (ms) | Mean (ms) | Median (ms) | SELECTs/request |
|---|---|---:|---:|---:|
| Per-book association queries (before) | 6.501, 7.218, 8.459, 7.167, 8.120 | 7.493 | 7.218 | 21 |
| Batched association preloads + joined category (after) | 2.843, 3.534, 4.154, 2.945, 2.845 | 3.264 | 2.945 | 3 |

These are local development measurements, not a production benchmark. With the warm-up request excluded consistently, the mean fell by about 56% and the median by about 59%.
