# Авитоша Backend — личный вклад Тимура Гилязова

Этот репозиторий содержит воспроизводимый срез моего личного backend-вклада в командный проект «Авитоша» для хакатона Avito Start. Я отвечал за фундамент Go-приложения, полный контур авторизации и управления сессиями, а также за retention- и reward-механики.

Часть командного игрового кода сохранена как необходимый контекст для сборки, тестирования и демонстрации интеграции. Точные границы авторства, исходные коммиты и состав среза описаны в [CONTRIBUTION.md](CONTRIBUTION.md).

Исходный командный репозиторий: [guitaramust-sudo/Avitosha](https://github.com/guitaramust-sudo/Avitosha).

## Моя зона ответственности

### Фундамент Go-backend

- конфигурация приложения из environment variables с валидацией;
- bootstrap и wiring монолитного HTTP API;
- структурированное логирование через `slog`;
- graceful shutdown и управление жизненным циклом сервиса;
- liveness/readiness endpoints;
- подключение к PostgreSQL через `pgxpool`;
- транзакционный менеджер с rollback при ошибке или panic;
- первоначальные Dockerfile, Docker Compose и локальное окружение;
- конфигурация `golangci-lint`.

### Авторизация и безопасность сессий

- регистрация, вход, refresh, logout и получение текущего пользователя;
- bcrypt для хранения паролей;
- короткоживущие JWT access tokens с `issuer`, `audience`, `subject` и `session_id`;
- криптографически случайные opaque refresh tokens;
- хранение только SHA-256-хешей refresh tokens;
- атомарная одноразовая ротация refresh token;
- отзыв сессий и поддержка нескольких сессий пользователя;
- серверная проверка актуальности сессии для каждого access token, поэтому logout инвалидирует и уже выданный JWT;
- HttpOnly/SameSite cookies с `Secure` в production;
- Bearer middleware, CORS, request ID, structured request logging и recovery middleware;
- единый публичный формат ошибок без утечки инфраструктурных деталей;
- OpenAPI-контракт и Swagger UI.

### Retention и награды

- серии ежедневной активности и milestone-награды;
- ежедневное задание с детерминированным назначением пользователю;
- прогноз следующего дня: ожидаемая серия, задание и награда;
- reward wallet, lifetime earned и каталог Avito-бонусов;
- отображение прогресса до следующего бонуса;
- интеграция streak и daily quest в общую транзакцию игрового действия;
- защита от повторного начисления через уникальный ledger источников награды;
- доменные события для обновления интерфейса;
- ограничения целостности и seed-данные в PostgreSQL-миграции.

## Архитектура среза

```mermaid
flowchart LR
    Client["HTTP / WebSocket client"] --> Handler["Handler: transport, validation, DTO"]
    Handler --> Usecase["Use case: auth, sessions, retention"]
    Usecase --> Repository["PostgreSQL repositories"]
    Repository --> DB[(PostgreSQL)]
    Usecase --> Events["Domain events after commit"]
```

Слои соединяются через небольшие интерфейсы. Репозитории получают активную транзакцию через `context.Context`, поэтому создание пользователя, сессии и стартового игрового профиля либо фиксируется целиком, либо целиком откатывается. Аналогично игровое действие, прогресс daily quest, streak и начисление наград выполняются в одной транзакционной границе.

## Как решения отвечают бизнес-задаче

Кейс требовал не только игровую механику, но и понятную причину вернуться завтра. Retention-контур делает эту причину явной:

- streak показывает ценность регулярности;
- daily quest связывает игровой прогресс с полезным действием на Авито;
- tomorrow preview заранее показывает следующий шаг и ожидаемую награду;
- reward wallet связывает активность с конкретными Avito-перками;
- progress до следующего бонуса объясняет, сколько осталось сделать;
- уникальный ledger не допускает повторного начисления одной награды.

## Локальный запуск

Потребуются Docker и Docker Compose.

```powershell
Copy-Item .env.example .env
docker compose up --build
```

После запуска:

- API: <http://localhost:8080>;
- Swagger UI: <http://localhost:8080/swagger/>;
- OpenAPI: <http://localhost:8080/api/openapi.yaml>;
- liveness: <http://localhost:8080/healthz>;
- readiness: <http://localhost:8080/health/ready>;
- PostgreSQL: `localhost:5433`.

Файл `.env.example` содержит только значения для локальной разработки. Перед production-запуском необходимо заменить `JWT_SIGNING_KEY`, настроить допустимый `FRONTEND_ORIGIN` и использовать секреты окружения.

Остановка стека:

```powershell
docker compose down
```

Для удаления локального Docker volume с данными используйте `docker compose down -v` только осознанно.

## Основные auth endpoints

| Метод | Endpoint | Назначение |
|---|---|---|
| `POST` | `/api/auth/register` | Регистрация и создание стартового профиля |
| `POST` | `/api/auth/login` | Вход и создание отдельной сессии |
| `POST` | `/api/auth/refresh` | Атомарная ротация refresh token |
| `POST` | `/api/auth/logout` | Отзыв текущей сессии |
| `GET` | `/api/me` | Текущий пользователь по Bearer token |
| `GET` | `/api/v1/daily-summary` | Дневной прогресс, streak, daily quest и tomorrow preview |
| `GET` | `/api/v1/rewards/balance` | Баланс и lifetime earned |
| `GET` | `/api/v1/rewards/wallet` | Каталог бонусов и следующая цель |

Полное описание запросов, ответов и ошибок находится в Swagger/OpenAPI.

## Проверка качества

```powershell
cd app/backend
go test ./...
go vet ./...
go mod verify
golangci-lint run
```

Repository и smoke-тесты PostgreSQL активируются только с безопасной тестовой базой, имя которой содержит `test`:

```powershell
$env:TEST_DATABASE_URL = "postgres://postgres:postgres@localhost:5433/avitosha_test?sslmode=disable"
go test ./internal/repository/postgres ./internal/handler
```

Тестами покрыты happy paths и критические отказы: rollback регистрации, duplicate email, неверный пароль, истекшая и отозванная сессия, повторное использование refresh token, немедленная инвалидизация access token после logout, внутренние ошибки без утечки деталей, CORS/recovery, конкурентная идемпотентность игровых действий и начислений.

## Ограничения MVP

- действия Авито поступают через mock API, а не через production event bus;
- каталог фиксирует достижение порога бонуса, но не выпускает настоящий промокод и не реализует redeem-flow;
- нет rate limiting, email verification, password reset и полноценного token-family reuse detection;
- WebSocket hub хранится в памяти и не рассчитан на несколько реплик;
- часть game core сохранена как командный код-контекст и не заявляется как моя индивидуальная реализация;
- значения секретов из `.env.example` предназначены только для локальной разработки.

## Использование ИИ

При подготовке этого личного репозитория OpenAI Codex использовался для анализа Git-истории, проверки границ авторства, сборки воспроизводимого среза и оформления документации. Исходный функциональный код взят из перечисленных в `CONTRIBUTION.md` хакатон-коммитов; кроме применения моего финального патча и адаптации упаковки backend-only, он не переписывался в процессе подготовки репозитория.

## Происхождение и права

Репозиторий подготовлен для индивидуальной оценки вклада в командный хакатон-проект. Командный контекст явно отмечен и не приписывается мне. Перед использованием этого кода вне целей оценки следует учитывать условия и права участников исходного проекта.
