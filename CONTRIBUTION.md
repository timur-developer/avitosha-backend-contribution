# Границы личного вклада (список созданных мной файлов)

## Полностью реализованные мной области

### Первоначальная инфраструктура

- `app/backend/cmd/api/main.go` в монолитной архитектуре;
- первоначальные `internal/config` и `internal/app/app.go`;
- `app/backend/internal/app/registration_hook.go` и тесты;
- `app/backend/api/spec.go`;
- `app/backend/migrations/000001_create_users.*`;
- `app/backend/migrations/000002_create_sessions.*`;
- `app/backend/.golangci.yml`;
- первоначальные backend Dockerfile, Compose и environment configuration.

### Auth и связанные модели

- `app/backend/internal/auth/**`;
- `app/backend/internal/model/user.go`;
- `app/backend/internal/model/session.go`;
- `app/backend/internal/model/authenticated_user.go`.

### Auth use cases и внутренние контракты

- `app/backend/internal/usecase/auth.go`;
- `app/backend/internal/usecase/session.go`;
- `app/backend/internal/usecase/password.go`;
- `app/backend/internal/usecase/token.go`;
- `app/backend/internal/usecase/user.go`;
- `app/backend/internal/usecase/tx.go`;
- `app/backend/internal/usecase/storage.go`;
- соответствующие `auth_test.go` и `session_auth_test.go`.

### PostgreSQL auth infrastructure

- `app/backend/internal/repository/postgres/user.go`;
- `app/backend/internal/repository/postgres/session.go`;
- `app/backend/internal/repository/postgres/tx.go`;
- `app/backend/internal/repository/postgres/pool.go`;
- связанные repository-тесты и test helpers.

### HTTP auth layer

- `app/backend/internal/handler/auth.go`;
- `app/backend/internal/handler/response.go`;
- основная реализация `middleware.go`;
- `app/backend/internal/handler/health.go`;
- `app/backend/internal/handler/swagger.go`;
- auth, health, Swagger и smoke-тесты.

### Мини-маркетплейс(классифайд) и игровая интеграция

- `app/backend/internal/model/listing.go`;
- `app/backend/internal/handler/marketplace.go` и `marketplace_error_test.go`;
- `app/backend/internal/usecase/marketplace.go`, `marketplace_test.go` и `marketplace_game_rules_test.go`;
- `app/backend/internal/repository/postgres/marketplace.go` и `marketplace_test.go`;
- `app/backend/migrations/000007_add_marketplace.*`;
- `app/backend/migrations/000008_add_marketplace_game_actions.*`;
- `app/backend/migrations/000009_make_demo_deals_user_scoped.*`;
- `app/backend/migrations/000010_add_first_room_furniture_seed.*`;
- `app/backend/migrations/000012_prevent_favorite_reward_replays.*`;
- `app/backend/migrations/marketplace_migration_test.go`;
- `app/backend/migrations/marketplace_game_actions_migration_test.go`;
- `app/backend/migrations/favorite_reward_migration_test.go`;
- `app/backend/internal/client/grpc/gameclient/marketplace_gateway_test.go`;
- `app/backend/internal/rpc/errors_test.go`;
- `docs/mini-avito-api.md`;
- `docs/mini-avito-implementation.md`.

В этой области я с нуля реализовал backend объявлений: каталог, собственные объявления, публикацию, избранное, уникальные просмотры, первое сообщение продавцу и demo-покупку. Также я реализовал серверную связь подтверждённых действий с игровой прогрессией, защиту от повторных начислений и тестовое покрытие ключевых сценариев. 


### Retention и reward wallet

- `app/backend/internal/model/retention.go`;
- `app/backend/internal/usecase/game_rewards_retention.go`;
- `app/backend/internal/repository/postgres/game_retention.go`;
- `app/backend/internal/repository/postgres/game_reward_catalog.go`;
- `app/backend/migrations/000006_add_reward_catalog_and_retention.up.sql`;
- `app/backend/migrations/000006_add_reward_catalog_and_retention.down.sql`;
- `app/backend/migrations/reward_catalog_retention_migration_test.go`.

### Production-деплой приложения в Yandex Cloud

- `.env.prod.example`;
- `compose.prod.yaml`;
- `deploy/Caddyfile`;
- `app/frontend/Dockerfile.prod`;
- `app/frontend/nginx.prod.conf`.

## Совместные файлы и мои изменения в них

| Файл                                                                                                                 | Мой вклад                                                                                                                       |
| -------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `internal/app/app.go`                                                                                                | Первоначальный bootstrap/lifecycle, PostgreSQL и auth wiring, подключение registration hook и server-side session authenticator |
| `internal/config/config.go`                                                                                          | Основная конфигурация backend; отдельные параметры позднее добавлялись командой                                                 |
| `internal/handler/router.go`                                                                                         | Auth routes/middleware и endpoint reward wallet                                                                                 |
| `internal/handler/game_identity.go`                                                                                  | Замена локальной проверки подписи JWT на проверку активной серверной сессии                                                     |
| `internal/handler/game.go`                                                                                           | `GetRewardWallet` и расширение контракта handler/use case                                                                       |
| `internal/handler/game_dto.go`                                                                                       | DTO и mapping для retention, daily quest, tomorrow preview, каталога и wallet                                                   |
| `internal/usecase/game_service.go`                                                                                   | Включение retention в общую транзакцию действия и daily summary; расширение reward ledger                                       |
| `internal/usecase/game_contracts.go`                                                                                 | Repository-контракты retention и каталога                                                                                       |
| `internal/usecase/game_errors.go`                                                                                    | Ошибка отсутствующего daily quest                                                                                               |
| `internal/repository/postgres/game_rewards.go`                                                                       | Источники наград и идемпотентный уникальный ledger                                                                              |
| `internal/model/game.go`                                                                                             | Reward source, reward credit metadata и catalog item                                                                            |
| `internal/model/domain_event.go`                                                                                     | События streak, daily quest и unlock каталога                                                                                   |
| `api/openapi.yaml`                                                                                                   | Auth-контракт и retention/reward endpoints/schemas                                                                              |
| `api/openapi_test.go`                                                                                                | Проверки auth и retention/reward контракта                                                                                      |
| `internal/usecase/game_service_test.go`                                                                              | Retention assertions и нормализация серии                                                                                       |
| `internal/repository/postgres/game_repository_test.go`                                                               | Проверка первого streak reward и отсутствия повторного начисления                                                               |
| `.gitignore` | Разрешение versioning шаблона `.env.prod.example`; защита от попадания в Git production-данных, дампов БД, сертификатов и ключей. |
| `app/frontend/.dockerignore` | Актуализация frontend Docker build context: исключение артефактов сборки, environment-файлов и лишних файлов. |
| `README.md` командного репозитория ([README.md](https://github.com/guitaramust-sudo/Avitosha/blob/master/README.md)) | Переработка структуры, навигации, технических разделов, инструкций по запуску и материалов для проверки; добавление раздела с публичным демо, ссылок на production-приложение и Swagger, а также описание реализованного production-деплоя в Yandex Cloud: отдельного Docker Compose-стека, nginx, Caddy, автоматического HTTPS, изоляции сервисов, persistent volumes, миграций и health check-ов. |
| `app/backend/api/openapi.yaml` | Добавил контракт Mini-Avito: DTO, маршруты и описания ошибок для объявлений, избранного, сообщений и demo-покупки. |
| `app/backend/api/proto/avitosha/v1/services.proto`, `internal/gen/avitosha/v1/services*.go` | Расширил внутренний gRPC-контракт, чтобы подтверждённое действие маркетплейса обрабатывалось игровым сервисом. |
| `internal/app/app.go`, `internal/app/microservices.go`, `internal/handler/router.go` | Подключил marketplace к приложению и маршрутам, а также связал gateway с game-service. |
| `internal/handler/game_identity.go`, `middleware.go` | Усилил идентификацию пользователя и защитил marketplace-сценарии авторизацией. |
| `internal/model/game.go`, `domain_event.go`, `internal/usecase/game_contracts.go`, `game_errors.go`, `game_service.go`, `game_rules.go` | Интегрировал marketplace actions в существующий игровой контур: XP, FIRST_ROOM, награды, достижения и события после commit. |
| `internal/repository/postgres/game_actions.go`, `game_progress.go`, `game_repository.go`, `error_mapping_test.go` | Добавил идемпотентность и целостность обработки действий, прогресса и ошибок marketplace. |


## Командный код-контекст

Для компиляции и проверки retention-интеграции сохранены компоненты, которые не являются полной моей реализацией:

- основное игровое ядро, XP, комнаты, сюжет, leaderboard, достижения и характер питомца (реализовывали мои коллеги Стас и Сергей);
- базовые game repositories и миграции `000003`–`000005`;
- WebSocket hub и публикация событий;
- последующие доработки UI мини-маркетплейса, загрузки файлов и MinIO, а также наборы ежедневных заданий - это совместная работа команды; они сохранены только как контекст, необходимый для запуска полного демо;
- smoke scripts и прочая инфраструктура, не перечисленная выше как мой вклад.

Эти файлы, чтобы запустить backend и увидеть работу функционала, реализованного мной (система вовлечения retention и расширенная информация в кошельке wallet reward) внутри реального пользовательского сценария.

### Документация проекта

Командный `README.md` является совместным файлом. Мой вклад включал существенную переработку его структуры, подготовку документации к технической проверке проекта и последующее документирование публичного production-деплоя:

- переработал структуру и оглавление;
- сгруппировал технические сведения в понятные разделы;
- улучшил представление инструкций по запуску и тестированию;
- актуализировал ссылки на дополнительную документацию;
- добавил технические детали, необходимые для оценки архитектуры и реализованного функционала;
- добавил раздел публичного демо, ссылку на развёрнутое приложение и Swagger-документацию;
- задокументировал в командном README реализованный production-деплой в Yandex Cloud и его ключевые технические характеристики.

Изменения:

- [`932bef8`](https://github.com/guitaramust-sudo/Avitosha/commit/932bef87902bb1d07d747e022af78a22bcd96e24) — основная переработка структуры README;
- [`953e655`](https://github.com/guitaramust-sudo/Avitosha/commit/953e655656037a550396eb2a5521a2b3026aedfe) — технические детали и актуализация навигации.
- [`d15c184`](https://github.com/guitaramust-sudo/Avitosha/commit/d15c18419c99540227d7badf07ec71281dd5644a) — production-конфигурация Yandex Cloud и первоначальное документирование деплоя;
- [`f6c5f9a`](https://github.com/guitaramust-sudo/Avitosha/commit/f6c5f9a6298d238adf25e5fa7bf47c00192836ef) — уточнение production-конфигурации и сохранение tracking миграций;
- [`0fda391`](https://github.com/guitaramust-sudo/Avitosha/commit/0fda391f3884198c19d6f27a9291f205b8971c34), [`29916c8`](https://github.com/guitaramust-sudo/Avitosha/commit/29916c8dd52c6056502df4edf53fed452e7a10a), [`df9da49`](https://github.com/guitaramust-sudo/Avitosha/commit/df9da499d7c87f8b3f500ba53e7747ff62149c4f) — актуализация описания демо, ссылок на развёрнутое приложение и Swagger-документацию, а также информации о вкладе в деплой.

## Ключевые коммиты по областям вклада

| Область                                 | Коммиты                                                                                                                                                                                                              |
| --------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Инициализация проекта и PostgreSQL       | [`bc6544d`](https://github.com/guitaramust-sudo/Avitosha/commit/bc6544de6912ebb2348fbd4013a65cd952176c68), [`63a0150`](https://github.com/guitaramust-sudo/Avitosha/commit/63a0150c4fbca96f0dd613063409798fe1af7454) |
| OpenAPI и Swagger                       | [`aee10bd`](https://github.com/guitaramust-sudo/Avitosha/commit/aee10bd56a3151e6978d2ceb7ebcc247adea2f62)                                                                                                            |
| Auth use cases и repositories           | [`bf91c4a`](https://github.com/guitaramust-sudo/Avitosha/commit/bf91c4a24a4484bc0fdaf409886e7f25b685e7a7)                                                                                                            |
| HTTP auth layer                         | [`36b8462`](https://github.com/guitaramust-sudo/Avitosha/commit/36b8462774d5373cef0b0b01cac7033ffa6e88c6)                                                                                                            |
| Backend linting                         | [`f255295`](https://github.com/guitaramust-sudo/Avitosha/commit/f255295bcc87f6fe902902cd9a2d85c43d41948c)                                                                                                            |
| Стартовый профиль при регистрации       | [`ae8bcd6`](https://github.com/guitaramust-sudo/Avitosha/commit/ae8bcd6b5780c7c253b6a6b362f3575543362de0)                                                                                                            |
| Retention и reward wallet               | [`d44f1b6`](https://github.com/guitaramust-sudo/Avitosha/commit/d44f1b66169f1d4e7249c3fc1dfe1945730d159a)                                                                                                            |
| Исправление reward transaction          | [`c4154d6`](https://github.com/guitaramust-sudo/Avitosha/commit/c4154d6d2c237aaee2c5d4323ecf59ac25c83b9e)                                                                                                            |
| Инвалидизация access token после logout | [`b8745ee`](https://github.com/guitaramust-sudo/Avitosha/commit/b8745eebd2646a982e623de6e1f564e1f73ad4c7)                                                                                                            |
| Финальная полировка                     | [`9245548`](https://github.com/guitaramust-sudo/Avitosha/commit/9245548296df9f6cb9c87bdd455b23080d5a38c5)                                                                                                            |
| Переработка структуры командного README | [`932bef8`](https://github.com/guitaramust-sudo/Avitosha/commit/932bef87902bb1d07d747e022af78a22bcd96e24)                                                                                                            |
| Технические детали и навигация README   | [`953e655`](https://github.com/guitaramust-sudo/Avitosha/commit/953e655656037a550396eb2a5521a2b3026aedfe)                                                                                                            |
| Production-деплой в Yandex Cloud | [`d15c184`](https://github.com/guitaramust-sudo/Avitosha/commit/d15c18419c99540227d7badf07ec71281dd5644a), [`f6c5f9a`](https://github.com/guitaramust-sudo/Avitosha/commit/f6c5f9a6298d238adf25e5fa7bf47c00192836ef) |
| Демо, Swagger и описание production-деплоя в README | [`0fda391`](https://github.com/guitaramust-sudo/Avitosha/commit/0fda391f3884198c19d6f27a9291f205b8971c34), [`29916c8`](https://github.com/guitaramust-sudo/Avitosha/commit/29916c8cdd52c6056502df4edf53fed452e7a10a), [`df9da49`](https://github.com/guitaramust-sudo/Avitosha/commit/df9da499d7c87f8b3f500ba53e7747ff62149c4f) |
| Mini-маркетплейс(классифайд) и связь с прогрессом питомца | [`d096166`](https://github.com/guitaramust-sudo/Avitosha/commit/d096166b414337fd815a49b7f6bc9eb6dd727661), [`4ae2ed3`](https://github.com/guitaramust-sudo/Avitosha/commit/4ae2ed3a9beb69518b70e815915b15efa96f1e06), [`5a3bcaa`](https://github.com/guitaramust-sudo/Avitosha/commit/5a3bcaa1f67fa5a30a544257c19d04c17e438148), [`8701b67`](https://github.com/guitaramust-sudo/Avitosha/commit/8701b676847a9f27ba54f93cac22438bc3bc40b1), [`6ae198d`](https://github.com/guitaramust-sudo/Avitosha/commit/6ae198d0d4b1c07f8b274403896ffbf6c4ed1f2b), [`634cfd8`](https://github.com/guitaramust-sudo/Avitosha/commit/634cfd8df62ed79d15b34c18b2aa5022aeab7fdb), [`62d4d34`](https://github.com/guitaramust-sudo/Avitosha/commit/62d4d340832ab1df8c98fc0ca4618282211c0c3e) |

Ссылка на мои коммиты в проекте: https://github.com/guitaramust-sudo/Avitosha/commits/master/?author=timur-developer
