# Mini-Avito API: этап 2

Публичны `GET /api/v1/listing-categories`, `GET /api/v1/listings` и `GET /api/v1/listings/{listing_id}`. Все действия требуют `Authorization: Bearer <token>` (в локальном режиме также поддержан `X-User-ID`). Карточка объявления всегда read-only: уникальный просмотр регистрируется только отдельным `POST`.

## Контракт мутаций

Создание черновика (`POST /api/v1/listings`) и снятие с публикации не имеют игрового эффекта. Следующие операции принимают клиентский UUID `eventId`; сервер сам определяет action type, категорию, XP, квесты и награды:

| Endpoint | Server action | Ответ |
| --- | --- | --- |
| `PATCH /api/v1/listings/{id}` | `LISTING_IMPROVED`, только для новых закрытых критериев опубликованного объявления | `listing`, опциональный `actionResult` |
| `POST /api/v1/listings/{id}/publish` | `AD_CREATED` | `listing`, `actionResult` |
| `PUT /api/v1/listings/{id}/favorite` | `AD_FAVORITED`, только при новом избранном | `listing`, `favorite`, `actionResult` |
| `POST /api/v1/listings/{id}/views` | `AD_VIEWED`, только один раз на пару пользователь–объявление–UTC-день | `listing`, `counted`, `actionResult` |
| `POST /api/v1/listings/{id}/messages` | `MESSAGE_SENT`, только при первом контакте | `listing`, `message`, `first`, `actionResult` |
| `POST /api/v1/listings/{id}/purchase` | `LISTING_SOLD`, дополнительно `DELIVERY_USED` при `deliveryUsed:true` | `listing`, `deal`, `actionResult` |

`actionResult` имеет форму `{ actionId, duplicate, events }`. Повтор того же события возвращает сохранённый результат с `duplicate:true`; XP, награды, задачи и WebSocket повторно не запускаются. Один `eventId` нельзя использовать для другого пользователя, объявления или операции. Для сделки с доставкой сервер детерминированно создаёт внутренний связанный event ID для `DELIVERY_USED`.

Повторная demo-покупка с **новым** `eventId` не создаёт вторую сделку: API отвечает `409 Conflict` с кодом `demo_purchase_already_completed`. Клиент должен показать, что покупка уже оформлена, и не трактовать ответ как временную серверную ошибку.

Качество `listing.quality` полностью серверное: цена больше нуля, хотя бы одно фото URL и описание от 150 символов. Для публикации обязательны непустое название и цена больше нуля; фото и подробное описание остаются рекомендациями качества и не блокируют публикацию. Поля: `score`, `isEligible`, `missingFields`, `nextActionHint`.

## Игровые эффекты

Базовые XP определены в `product_action_rules`: просмотр 2, избранное 5, сообщение 8, качественная публикация 40, улучшение 20, продажа 50, доставка 15. Награда завершённого существующего квеста начисляется дополнительно.

Все изменения marketplace и игрового контура выполняются в одной PostgreSQL-транзакции. После commit пользователю отправляется существующий WebSocket envelope `{ "events": [...] }`; среди событий могут быть `LISTING_VIEWED`, `LISTING_FAVORITED`, `SELLER_CONTACTED`, `LISTING_PUBLISHED`, `LISTING_IMPROVED`, `LISTING_SOLD`, `DELIVERY_USED` вместе с `XP_EARNED`, task, pet, story и reward-событиями. События изолированы по пользователю и не публикуются при rollback.

`FIRST_ROOM` использует реальные операции: пять уникальных просмотров мебели → избранное мебели → первое сообщение продавцу → качественная публикация → demo-сделка с доставкой. Идентификаторы и порядок заданий не изменены.

## Seed-сценарий

UUID demo-объявлений: `10000000-0000-0000-0000-000000000001`–`...0004`. Первые два относятся к `FURNITURE`: ими проходят просмотр, избранное и сообщение. Любое опубликованное demo-объявление подходит для однократной покупки с доставкой. Пользователь создаёт собственный качественный черновик, затем публикует его для этапа 4 FIRST_ROOM.

Полный формальный контракт, ошибки, пагинация, примеры и схемы находятся в [`app/backend/api/openapi.yaml`](../app/backend/api/openapi.yaml). Старый `POST /api/v1/actions` сохранён для обратной совместимости.
