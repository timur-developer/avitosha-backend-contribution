ALTER TABLE reward_transactions
    ALTER COLUMN task_id DROP NOT NULL;

ALTER TABLE reward_transactions
    ADD COLUMN source_kind TEXT NOT NULL DEFAULT 'TASK_COMPLETION',
    ADD COLUMN source_ref TEXT,
    ADD COLUMN source_title TEXT;

UPDATE reward_transactions
SET source_ref = task_id::TEXT
WHERE source_ref IS NULL;

ALTER TABLE reward_transactions
    ALTER COLUMN source_ref SET NOT NULL;

ALTER TABLE reward_transactions
    DROP CONSTRAINT IF EXISTS reward_transactions_action_id_task_id_reward_type_key;

CREATE UNIQUE INDEX idx_reward_transactions_source
    ON reward_transactions (action_id, reward_type, source_kind, source_ref);

ALTER TABLE reward_transactions
    ADD CONSTRAINT reward_transactions_source_kind_check
    CHECK (source_kind IN ('TASK_COMPLETION', 'DAILY_QUEST', 'STREAK'));

CREATE TABLE reward_catalog_items (
    code TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    reward_type TEXT NOT NULL,
    perk_type TEXT NOT NULL,
    threshold BIGINT NOT NULL,
    sort_order SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (btrim(code) <> ''),
    CHECK (btrim(title) <> ''),
    CHECK (btrim(reward_type) <> ''),
    CHECK (btrim(perk_type) <> ''),
    CHECK (threshold > 0),
    CHECK (sort_order >= 0),
    CHECK (updated_at >= created_at)
);

CREATE TABLE user_streaks (
    user_id UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    current_streak INTEGER NOT NULL DEFAULT 0,
    longest_streak INTEGER NOT NULL DEFAULT 0,
    last_active_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (current_streak >= 0),
    CHECK (longest_streak >= current_streak),
    CHECK (updated_at >= created_at)
);

CREATE TABLE daily_quest_templates (
    code TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    action_type TEXT NOT NULL,
    category TEXT,
    target_value INTEGER NOT NULL,
    reward_type TEXT NOT NULL,
    reward_amount INTEGER NOT NULL,
    sort_order SMALLINT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (btrim(code) <> ''),
    CHECK (btrim(title) <> ''),
    CHECK (action_type IN ('AD_VIEWED', 'AD_FAVORITED', 'MESSAGE_SENT', 'AD_CREATED', 'DELIVERY_USED', 'REVIEW_LEFT', 'BOOKING_CREATED')),
    CHECK (target_value > 0),
    CHECK (btrim(reward_type) <> ''),
    CHECK (reward_amount > 0),
    CHECK (sort_order >= 0),
    CHECK (updated_at >= created_at)
);

CREATE TABLE user_daily_quests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    quest_date DATE NOT NULL,
    template_code TEXT NOT NULL REFERENCES daily_quest_templates (code),
    progress INTEGER NOT NULL DEFAULT 0,
    target_value INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    reward_type TEXT NOT NULL,
    reward_amount INTEGER NOT NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    rewarded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, quest_date),
    CHECK (progress BETWEEN 0 AND target_value),
    CHECK (target_value > 0),
    CHECK (status IN ('ACTIVE', 'COMPLETED', 'REWARDED', 'EXPIRED')),
    CHECK ((status IN ('COMPLETED', 'REWARDED')) = (completed_at IS NOT NULL)),
    CHECK ((status = 'REWARDED') = (rewarded_at IS NOT NULL)),
    CHECK (reward_amount > 0),
    CHECK (updated_at >= created_at)
);

CREATE INDEX idx_user_daily_quests_lookup
    ON user_daily_quests (user_id, quest_date DESC);

ALTER TABLE domain_events
    DROP CONSTRAINT IF EXISTS domain_events_event_type_check;

ALTER TABLE domain_events
    ADD CONSTRAINT domain_events_event_type_check
    CHECK (
        event_type IN (
            'TASK_PROGRESS_UPDATED',
            'TASK_COMPLETED',
            'XP_EARNED',
            'PET_LEVEL_UP',
            'PET_MOOD_CHANGED',
            'ROOM_ITEM_UNLOCKED',
            'STORY_STAGE_COMPLETED',
            'STORY_COMPLETED',
            'LEADERBOARD_SCORE_UPDATED',
            'ACHIEVEMENT_UNLOCKED',
            'PET_CHARACTER_UNLOCKED',
            'AVITO_REWARD_EARNED',
            'REWARD_CATALOG_UNLOCKED',
            'DAILY_QUEST_UPDATED',
            'DAILY_QUEST_COMPLETED',
            'STREAK_UPDATED'
        )
    );

INSERT INTO reward_catalog_items (code, title, description, reward_type, perk_type, threshold, sort_order) VALUES
    ('FREE_DELIVERY_LIGHT', 'Бесплатная доставка', 'Персональный бонус на одну доставку через Авито Доставку.', 'AVITO_BONUS', 'DELIVERY', 20, 1),
    ('CATEGORY_DISCOUNT_HOME', 'Скидка в категории', 'Скидка на услуги продвижения для товаров дома и мебели.', 'AVITO_BONUS', 'CATEGORY_DISCOUNT', 45, 2),
    ('PROMO_BOOST', 'Бонус на продвижение', 'Небольшой пакет бонусов на продвижение объявления.', 'AVITO_BONUS', 'PROMOTION', 75, 3),
    ('AUTOTEKA_CHECK', 'Проверка Автотеки', 'Одна бесплатная проверка истории автомобиля.', 'AVITO_BONUS', 'AUTOTEKA', 110, 4),
    ('SELLER_LIMIT_PACK', 'Повышенный лимит', 'Небольшое расширение лимита на размещение и продвижение.', 'AVITO_BONUS', 'LIMIT_PACK', 160, 5);

INSERT INTO daily_quest_templates (
    code, title, description, action_type, category, target_value, reward_type, reward_amount, sort_order
) VALUES
    ('DAILY_DISCOVER', 'Найди что-то интересное', 'Посмотри 3 объявления и собери идеи для новых покупок.', 'AD_VIEWED', NULL, 3, 'AVITO_BONUS', 5, 1),
    ('DAILY_FAVORITE', 'Сохрани находку', 'Добавь одно объявление в избранное.', 'AD_FAVORITED', NULL, 1, 'AVITO_BONUS', 6, 2),
    ('DAILY_CONTACT', 'Уточни детали', 'Напиши продавцу и узнай подробности.', 'MESSAGE_SENT', NULL, 1, 'AVITO_BONUS', 8, 3),
    ('DAILY_SELLER_STEP', 'Сделай шаг продавца', 'Размести одно объявление или вернись к продаже.', 'AD_CREATED', NULL, 1, 'AVITO_BONUS', 10, 4),
    ('DAILY_DELIVERY', 'Проверь доставку', 'Используй Авито Доставку и закрой полезный шаг дня.', 'DELIVERY_USED', NULL, 1, 'AVITO_BONUS', 12, 5);
