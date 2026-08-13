ALTER TABLE daily_quest_templates
    ADD COLUMN role TEXT,
    ADD COLUMN xp_reward INTEGER NOT NULL DEFAULT 10;

UPDATE daily_quest_templates SET role = CASE
    WHEN action_type IN ('AD_VIEWED', 'AD_FAVORITED', 'MESSAGE_SENT') THEN 'BUYER'
    WHEN action_type IN ('AD_CREATED', 'LISTING_IMPROVED', 'LISTING_SOLD') THEN 'SELLER'
    ELSE 'UNIVERSAL'
END;

ALTER TABLE daily_quest_templates
    ALTER COLUMN role SET NOT NULL,
    ADD CONSTRAINT daily_quest_templates_role_check CHECK (role IN ('BUYER', 'SELLER', 'UNIVERSAL')),
    ADD CONSTRAINT daily_quest_templates_xp_reward_check CHECK (xp_reward > 0);

ALTER TABLE user_streaks ADD COLUMN protection_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE user_streaks ADD CONSTRAINT user_streaks_protection_count_check CHECK (protection_count >= 0);

ALTER TABLE daily_quest_templates DROP CONSTRAINT IF EXISTS daily_quest_templates_action_type_check;
ALTER TABLE daily_quest_templates ADD CONSTRAINT daily_quest_templates_action_type_check
    CHECK (action_type IN (
        'AD_VIEWED', 'AD_FAVORITED', 'MESSAGE_SENT', 'AD_CREATED', 'DELIVERY_USED',
        'REVIEW_LEFT', 'BOOKING_CREATED', 'LISTING_IMPROVED', 'LISTING_SOLD'
    ));

ALTER TABLE daily_quest_templates DROP CONSTRAINT IF EXISTS daily_quest_templates_reward_amount_check;
ALTER TABLE daily_quest_templates ADD CONSTRAINT daily_quest_templates_reward_amount_check CHECK (reward_amount >= 0);

UPDATE daily_quest_templates SET reward_amount = 0;

INSERT INTO daily_quest_templates (
    code, title, description, action_type, role, category, target_value,
    xp_reward, reward_type, reward_amount, sort_order
) VALUES
    ('DAILY_COMPARE', 'Сравни варианты', 'Посмотри 5 разных объявлений.', 'AD_VIEWED', 'BUYER', NULL, 5, 8, 'AVITO_BONUS', 0, 6),
    ('DAILY_DIALOG', 'Начни диалог', 'Напиши продавцу по интересному объявлению.', 'MESSAGE_SENT', 'BUYER', NULL, 1, 12, 'AVITO_BONUS', 0, 7),
    ('DAILY_NEW_LISTING', 'Новое объявление', 'Создай новое объявление.', 'AD_CREATED', 'SELLER', NULL, 1, 15, 'AVITO_BONUS', 0, 8),
    ('DAILY_IMPROVE_LISTING', 'Улучши объявление', 'Улучши описание, цену или фотографии объявления.', 'LISTING_IMPROVED', 'SELLER', NULL, 1, 12, 'AVITO_BONUS', 0, 9)
ON CONFLICT (code) DO NOTHING;

ALTER TABLE user_daily_quests DROP CONSTRAINT IF EXISTS user_daily_quests_user_id_quest_date_key;
ALTER TABLE user_daily_quests ADD CONSTRAINT user_daily_quests_user_date_template_key
    UNIQUE (user_id, quest_date, template_code);
ALTER TABLE user_daily_quests DROP CONSTRAINT IF EXISTS user_daily_quests_reward_amount_check;
ALTER TABLE user_daily_quests ADD CONSTRAINT user_daily_quests_reward_amount_check CHECK (reward_amount >= 0);

CREATE TABLE user_daily_goals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    goal_date DATE NOT NULL,
    required_completed INTEGER NOT NULL DEFAULT 2,
    completed_count INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    xp_reward INTEGER NOT NULL DEFAULT 30,
    reward_type TEXT NOT NULL DEFAULT 'AVITO_BONUS',
    reward_amount INTEGER NOT NULL DEFAULT 5,
    balanced_reward_amount INTEGER NOT NULL DEFAULT 3,
    rewarded_at TIMESTAMPTZ,
    balanced_rewarded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, goal_date),
    CHECK (required_completed > 0),
    CHECK (completed_count >= 0),
    CHECK (status IN ('ACTIVE', 'REWARDED', 'EXPIRED')),
    CHECK ((status = 'REWARDED') = (rewarded_at IS NOT NULL)),
    CHECK (xp_reward > 0),
    CHECK (reward_amount > 0),
    CHECK (balanced_reward_amount > 0),
    CHECK (updated_at >= created_at)
);

CREATE INDEX idx_user_daily_goals_lookup ON user_daily_goals (user_id, goal_date DESC);

ALTER TABLE reward_transactions DROP CONSTRAINT IF EXISTS reward_transactions_source_kind_check;
ALTER TABLE reward_transactions ADD CONSTRAINT reward_transactions_source_kind_check
    CHECK (source_kind IN ('TASK_COMPLETION', 'DAILY_QUEST', 'DAILY_GOAL', 'BALANCED_DAY', 'STREAK'));

ALTER TABLE domain_events DROP CONSTRAINT IF EXISTS domain_events_event_type_check;
ALTER TABLE domain_events ADD CONSTRAINT domain_events_event_type_check CHECK (event_type IN (
    'TASK_PROGRESS_UPDATED', 'TASK_COMPLETED', 'XP_EARNED', 'PET_LEVEL_UP', 'PET_MOOD_CHANGED',
    'ROOM_ITEM_UNLOCKED', 'STORY_STAGE_COMPLETED', 'STORY_COMPLETED', 'LEADERBOARD_SCORE_UPDATED',
    'ACHIEVEMENT_UNLOCKED', 'PET_CHARACTER_UNLOCKED', 'AVITO_REWARD_EARNED',
    'REWARD_CATALOG_UNLOCKED', 'DAILY_QUEST_UPDATED', 'DAILY_QUEST_COMPLETED',
    'DAILY_GOAL_COMPLETED', 'BALANCED_DAY_COMPLETED', 'STREAK_UPDATED', 'LISTING_VIEWED',
    'LISTING_FAVORITED', 'SELLER_CONTACTED', 'LISTING_PUBLISHED', 'LISTING_IMPROVED',
    'LISTING_SOLD', 'DELIVERY_USED'
));
